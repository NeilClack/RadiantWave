#version 330 core

layout (location = 0) in vec2 in_Position;
layout (location = 1) in vec2 in_TexCoord;
out vec2 pass_TexCoord;
uniform mat4 u_mvpMatrix;

void main()
{
    gl_Position = u_mvpMatrix * vec4(in_Position.x, in_Position.y, 0.0, 1.0);
    pass_TexCoord = in_TexCoord;
}
