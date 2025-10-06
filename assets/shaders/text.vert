#version 330 core

// Input vertex data from the VBO
layout (location = 0) in vec2 aPos;      // The (x, y) position of the vertex
layout (location = 1) in vec2 aTexCoord; // The (u, v) texture coordinate for this vertex

// Output to the fragment shader
out vec2 TexCoord;

// A matrix to handle projection (not strictly needed for this specific render-to-texture,
// but good practice to include for general purpose use)
uniform mat4 u_mvpMatrix;

void main()
{
    // Set the final position of the vertex
    gl_Position = u_mvpMatrix * vec4(aPos.x, aPos.y, 0.0, 1.0);

    // Pass the texture coordinate straight through to the fragment shader
    TexCoord = aTexCoord;
}
