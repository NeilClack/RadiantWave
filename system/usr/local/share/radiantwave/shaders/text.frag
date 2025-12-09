#version 330 core

// The final color output for the pixel
out vec4 FragColor;

// Input from the vertex shader
in vec2 TexCoord; // The texture coordinate, interpolated for this specific pixel

// The source texture (our FontMap "sticker sheet")
uniform sampler2D u_texture;
uniform vec4 u_textColor;

void main()
{
    float mask = texture(u_texture, TexCoord).a;
    FragColor = vec4(u_textColor.rgb, u_textColor.a * mask);
}
