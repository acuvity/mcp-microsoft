FROM alpine:3.21

COPY mcp-server-microsoft-graph /mcp-server-microsoft-graph

EXPOSE 8000

ENTRYPOINT [ "/mcp-server-microsoft-graph" ]
