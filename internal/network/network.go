// Package network provides a high-level utility for managing network
// connections via the system's D-Bus and NetworkManager.
package network

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

// --- D-Bus NetworkManager Constants ---
const (
	nmService                 = "org.freedesktop.NetworkManager"
	nmPath                    = "/org/freedesktop/NetworkManager"
	nmInterface               = "org.freedesktop.NetworkManager"
	nmDeviceInterface         = "org.freedesktop.NetworkManager.Device"
	nmWirelessDeviceInterface = "org.freedesktop.NetworkManager.Device.Wireless"
	nmAPInterface             = "org.freedesktop.NetworkManager.AccessPoint"

	nmDeviceTypeEthernet   uint32 = 1
	nmDeviceTypeWifi       uint32 = 2
	nmDeviceStateActivated uint32 = 100
	APFlagsPrivacy         uint32 = 0x00000001 // Equivalent to NM_802_11_AP_FLAGS_PRIVACY
	APSecKeyMgmtEAP        uint32 = 0x00000020
	APSecKeyMgmtPSK        uint32 = 0x00000100
	APSecKeyMgmtOWE        uint32 = 0x00000200
	APSecKeyMgmtSAE        uint32 = 0x00000400
)

// ErrScanAborted is returned when a scan is cancelled via the stop channel.
var ErrScanAborted = errors.New("network scan aborted by caller")

// Status defines the type for various connection status codes.
type Status int

const (
	StatusError Status = iota
	StatusNoDevicesFound
	StatusWifiAvailableNotConnected
	StatusWifiConnectedNoInternet
	StatusWifiConnectedInternetUp
	StatusEthernetNotConnected
	StatusEthernetConnectedNoInternet
	StatusEthernetConnectedInternetUp
)

// ConnectionStatus holds the detailed result of a connectivity check.
type ConnectionStatus struct {
	StatusCode Status
	Message    string
}

// DeviceType defines the kind of network device.
type DeviceType int

const (
	TypeUnknown DeviceType = iota
	TypeEthernet
	TypeWifi
)

// Device holds information about a single network device.
type Device struct {
	Path          dbus.ObjectPath
	InterfaceName string
	Type          DeviceType
	State         uint32
}

// Manager handles all D-Bus communication with NetworkManager.
type Manager struct {
	conn     *dbus.Conn
	nmObject dbus.BusObject
	Devices  []Device
}

// AccessPoint holds essential information about a discovered Wi-Fi network.
type AccessPoint struct {
	ObjectPath  dbus.ObjectPath
	SSID        string
	Strength    byte
	IsProtected bool
	IsActive    bool
	WpaFlags    uint32
	RsnFlags    uint32
}

// New creates a new network Manager and connects to the system D-Bus.
func New() (*Manager, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}
	nmObject := conn.Object(nmService, nmPath)
	return &Manager{conn: conn, nmObject: nmObject}, nil
}

// Conn returns the underlying dbus.Conn for advanced queries.
func (m *Manager) Conn() *dbus.Conn {
	return m.conn
}

// Close disconnects from the D-Bus.
func (m *Manager) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
}

// FindDevices scans the system for all available network devices (Ethernet and Wi-Fi).
func (m *Manager) FindDevices() error {
	var devicePaths []dbus.ObjectPath
	call := m.nmObject.Call(nmInterface+".GetDevices", 0)
	if call.Err != nil {
		return fmt.Errorf("GetDevices D-Bus call failed: %w", call.Err)
	}
	if err := call.Store(&devicePaths); err != nil {
		return fmt.Errorf("could not parse device list: %w", err)
	}

	m.Devices = []Device{}
	for _, path := range devicePaths {
		deviceObj := m.conn.Object(nmService, path)
		typeVariant, err := deviceObj.GetProperty(nmDeviceInterface + ".DeviceType")
		if err != nil {
			log.Printf("network: Skipping device %s, could not get DeviceType: %v", path, err)
			continue
		}
		deviceType, _ := typeVariant.Value().(uint32)

		var newDevice Device
		switch deviceType {
		case nmDeviceTypeEthernet:
			newDevice.Type = TypeEthernet
		case nmDeviceTypeWifi:
			newDevice.Type = TypeWifi
		default:
			continue
		}

		newDevice.Path = path
		ifaceVariant, _ := deviceObj.GetProperty(nmDeviceInterface + ".Interface")
		newDevice.InterfaceName, _ = ifaceVariant.Value().(string)
		stateVariant, _ := deviceObj.GetProperty(nmDeviceInterface + ".State")
		newDevice.State, _ = stateVariant.Value().(uint32)
		m.Devices = append(m.Devices, newDevice)
	}
	return nil
}

// CheckInternetConnection performs a comprehensive check for an active internet connection.
func (m *Manager) CheckInternetConnection() (*ConnectionStatus, error) {
	if err := m.FindDevices(); err != nil {
		return &ConnectionStatus{StatusError, "Failed to scan for network devices"}, err
	}
	if len(m.Devices) == 0 {
		return &ConnectionStatus{StatusNoDevicesFound, "No network hardware found."}, nil
	}

	hasWifiAdapter := false
	hasEthernetAdapter := false

	for _, device := range m.Devices {
		if device.Type == TypeEthernet {
			hasEthernetAdapter = true
			if device.State == nmDeviceStateActivated {
				if pingTest() {
					return &ConnectionStatus{StatusEthernetConnectedInternetUp, "Ethernet connected with internet access."}, nil
				}
				return &ConnectionStatus{StatusEthernetConnectedNoInternet, "Ethernet connected, but no internet access."}, nil
			}
		}
	}

	for _, device := range m.Devices {
		if device.Type == TypeWifi {
			hasWifiAdapter = true
			if device.State == nmDeviceStateActivated {
				if pingTest() {
					return &ConnectionStatus{StatusWifiConnectedInternetUp, "Wi-Fi connected with internet access."}, nil
				}
				return &ConnectionStatus{StatusWifiConnectedNoInternet, "Wi-Fi connected, but no internet access."}, nil
			}
		}
	}

	if hasWifiAdapter {
		return &ConnectionStatus{StatusWifiAvailableNotConnected, "Wi-Fi adapter is available but not connected."}, nil
	}
	if hasEthernetAdapter {
		return &ConnectionStatus{StatusEthernetNotConnected, "Ethernet adapter is available but not connected."}, nil
	}

	return &ConnectionStatus{StatusNoDevicesFound, "No recognizable network hardware found."}, nil
}

// StartScan performs a Wi-Fi scan asynchronously. The optional stopChan can be used to abort.
func (m *Manager) StartScan(resultsChan chan<- []AccessPoint, errorChan chan<- error, stopChan <-chan struct{}) {
	var wifiDevice *Device
	for i := range m.Devices {
		if m.Devices[i].Type == TypeWifi {
			wifiDevice = &m.Devices[i]
			break
		}
	}
	if wifiDevice == nil {
		errorChan <- fmt.Errorf("no Wi-Fi device available for scanning")
		return
	}

	// Ensure Wi-Fi is enabled
	nmObj := m.conn.Object(nmService, nmPath)
	prop, err := nmObj.GetProperty(nmInterface + ".WirelessEnabled")
	if err != nil {
		errorChan <- fmt.Errorf("could not read WirelessEnabled property: %w", err)
		return
	}
	if !prop.Value().(bool) {
		call := nmObj.Call("org.freedesktop.DBus.Properties.Set", 0,
			nmInterface,
			"WirelessEnabled",
			dbus.MakeVariant(true),
		)
		if call.Err != nil {
			errorChan <- fmt.Errorf("could not enable Wi-Fi: %w", call.Err)
			return
		}
		time.Sleep(2 * time.Second) // Give NetworkManager time to bring up the device
	}

	// Request a scan
	wifiDeviceWireless := m.conn.Object(nmService, wifiDevice.Path)
	call := wifiDeviceWireless.Call(nmWirelessDeviceInterface+".RequestScan", 0, make(map[string]dbus.Variant))
	if call.Err != nil {
		errorChan <- fmt.Errorf("scan request failed: %w", call.Err)
		return
	}

	select {
	case <-time.After(3 * time.Second):
	case <-stopChan:
		errorChan <- ErrScanAborted
		return
	}

	var apPaths []dbus.ObjectPath
	call = wifiDeviceWireless.Call(nmWirelessDeviceInterface+".GetAllAccessPoints", 0)
	if call.Err != nil {
		errorChan <- fmt.Errorf("could not retrieve network list: %w", call.Err)
		return
	}
	if err := call.Store(&apPaths); err != nil {
		errorChan <- fmt.Errorf("could not parse network list: %w", err)
		return
	}

	processedAPs, err := m.scanAccessPoints(apPaths)
	if err != nil {
		errorChan <- err
		return
	}
	resultsChan <- processedAPs
}

// scanAccessPoints filters and sorts the raw list of D-Bus AP paths.
func (m *Manager) scanAccessPoints(apPaths []dbus.ObjectPath) ([]AccessPoint, error) {
	tempNetworkMap := make(map[string]AccessPoint)
	for _, apPath := range apPaths {
		apObj := m.conn.Object(nmService, apPath)
		props, err := m.getAllProperties(apObj, nmAPInterface)
		if err != nil {
			log.Printf("network: Failed to get properties for AP %s: %v", apPath, err)
			continue
		}

		ssidBytes, _ := props["Ssid"].Value().([]byte)
		if len(ssidBytes) == 0 {
			continue
		}

		ssidStr := string(ssidBytes)
		strength, _ := props["Strength"].Value().(byte)
		flags, _ := props["Flags"].Value().(uint32)
		wpaFlags, _ := props["WpaFlags"].Value().(uint32)
		rsnFlags, _ := props["RsnFlags"].Value().(uint32)

		isProtected := (flags&APFlagsPrivacy != 0) || (wpaFlags != 0) || (rsnFlags != 0)

		currentAP := AccessPoint{
			ObjectPath:  apPath,
			SSID:        ssidStr,
			Strength:    strength,
			IsProtected: isProtected,
			WpaFlags:    wpaFlags,
			RsnFlags:    rsnFlags,
		}
		if existingAP, ok := tempNetworkMap[ssidStr]; !ok || strength > existingAP.Strength {
			tempNetworkMap[ssidStr] = currentAP
		}
	}

	networkList := make([]AccessPoint, 0, len(tempNetworkMap))
	for _, ap := range tempNetworkMap {
		networkList = append(networkList, ap)
	}
	sort.Slice(networkList, func(i, j int) bool {
		if networkList[i].Strength != networkList[j].Strength {
			return networkList[i].Strength > networkList[j].Strength
		}
		return strings.ToLower(networkList[i].SSID) < strings.ToLower(networkList[j].SSID)
	})
	return networkList, nil
}

// Connect attempts to connect a Wi-Fi device to a given access point.
func (m *Manager) Connect(wifiDevice Device, ap AccessPoint, passphrase string) error {
	if wifiDevice.Type != TypeWifi {
		return fmt.Errorf("cannot connect: provided device is not a Wi-Fi device")
	}

	settings := map[string]map[string]dbus.Variant{
		"connection": {
			"type":           dbus.MakeVariant("802-11-wireless"),
			"uuid":           dbus.MakeVariant(uuid.NewString()),
			"id":             dbus.MakeVariant(ap.SSID),
			"interface-name": dbus.MakeVariant(wifiDevice.InterfaceName),
		},
		"802-11-wireless": {
			"ssid": dbus.MakeVariant([]byte(ap.SSID)),
			"mode": dbus.MakeVariant("infrastructure"),
		},
	}

	if ap.IsProtected {
		settings["802-11-wireless"]["security"] = dbus.MakeVariant("802-11-wireless-security")
		secSettings := make(map[string]dbus.Variant)

		// Default to WPA-PSK, but override with more specific types if found.
		secSettings["key-mgmt"] = dbus.MakeVariant("wpa-psk")
		if (ap.RsnFlags & APSecKeyMgmtSAE) != 0 {
			secSettings["key-mgmt"] = dbus.MakeVariant("sae") // WPA3
		} else if (ap.RsnFlags&APSecKeyMgmtEAP) != 0 || (ap.WpaFlags&APSecKeyMgmtEAP) != 0 {
			log.Println("network: Enterprise AP (EAP) detected. Full configuration may be required.")
			secSettings["key-mgmt"] = dbus.MakeVariant("wpa-eap")
		}

		secSettings["psk"] = dbus.MakeVariant(passphrase)
		settings["802-11-wireless-security"] = secSettings
	}

	call := m.nmObject.Call(nmInterface+".AddAndActivateConnection", 0, settings, wifiDevice.Path, ap.ObjectPath)
	if call.Err != nil {
		return fmt.Errorf("AddAndActivateConnection D-Bus call error: %w", call.Err)
	}
	return nil
}

// pingTest performs a simple DNS lookup to check for internet connectivity.
func pingTest() bool {
	_, err := net.LookupHost("www.google.com")
	return err == nil
}

// getAllProperties is a helper to fetch all properties for a D-Bus object interface.
func (m *Manager) getAllProperties(obj dbus.BusObject, iface string) (map[string]dbus.Variant, error) {
	var props map[string]dbus.Variant
	call := obj.Call("org.freedesktop.DBus.Properties.GetAll", 0, iface)
	if call.Err != nil {
		return nil, call.Err
	}
	if err := call.Store(&props); err != nil {
		return nil, err
	}
	return props, nil
}

// WaitForInternet blocks until CheckInternetConnection reports Internet up,
// or until maxWait elapses. When it first sees "up", it requires that state
// to be stable for at least 'stableFor' before returning success to avoid flapping.
func (m *Manager) WaitForInternet(maxWait, stableFor time.Duration) (*ConnectionStatus, error) {
	deadline := time.Now().Add(maxWait)
	var lastErr error
	var lastStatus *ConnectionStatus
	var upSince time.Time

	// Reasonable poll cadence; NM typically completes within a few seconds.
	const poll = 500 * time.Millisecond

	for time.Now().Before(deadline) {
		st, err := m.CheckInternetConnection()
		if err != nil {
			lastErr = err
			lastStatus = &ConnectionStatus{StatusError, "NetworkManager not ready"}
			upSince = time.Time{} // reset stability
		} else {
			lastStatus = st
			switch st.StatusCode {
			case StatusEthernetConnectedInternetUp, StatusWifiConnectedInternetUp:
				if upSince.IsZero() {
					upSince = time.Now()
				}
				if time.Since(upSince) >= stableFor {
					return st, nil // stable "up"
				}
			default:
				// not up; keep waiting
				upSince = time.Time{}
			}
		}
		time.Sleep(poll)
	}

	// Timed out; return the last observed status and error (if any).
	if lastStatus == nil {
		lastStatus = &ConnectionStatus{StatusError, "Timed out waiting for connectivity"}
	}
	if lastErr != nil {
		return lastStatus, fmt.Errorf("timeout waiting for internet: %w", lastErr)
	}
	return lastStatus, nil
}
