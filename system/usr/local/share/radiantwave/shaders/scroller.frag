#version 330 core

in vec2 pass_TexCoord; 
out vec4 out_Color;
uniform sampler2D u_fontTexture;
uniform vec4 u_textColor;

void main()
{
    vec4 texel = texture(u_fontTexture, pass_TexCoord);
    float mask = texel.a;
    out_Color = vec4(u_textColor.rgb, u_textColor.a * mask);
}
