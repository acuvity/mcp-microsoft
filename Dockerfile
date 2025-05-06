FROM alpine:3.21

COPY mcp-microsoft /mcp-microsoft

EXPOSE 8000

ENTRYPOINT [ "/mcp-microsoft" ]
