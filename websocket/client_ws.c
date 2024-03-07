#include <libwebsockets.h>
#include <string.h>
#include <signal.h>
#include <stdio.h>
#include <unistd.h>

static int force_exit = 0;
static struct lws *wsi_global = NULL;

void sighandler(int sig)
{
    force_exit = 1;
}

void send_ws_text_message(struct lws *wsi, const char *message) {
    if (wsi) {
        size_t message_len = strlen(message);

        // Calculate total message size including LWS_PRE
        size_t total_len = LWS_PRE + message_len;

        // Allocate memory for the message with LWS_PRE bytes before the actual message
        unsigned char *buffer = (unsigned char *)malloc(total_len);
        if (buffer) {
            // Copy message into the buffer after the LWS_PRE bytes
            memcpy(buffer + LWS_PRE, message, message_len);

            // Send the message
            lws_write(wsi, buffer + LWS_PRE, message_len, LWS_WRITE_TEXT);

            // Free the allocated buffer
            free(buffer);
        } else {
            // Handle memory allocation failure
            fprintf(stderr, "Failed to allocate memory for message\n");
        }
    } else {
        fprintf(stderr, "WebSocket instance is NULL\n");
    }
}

static int callback_client(struct lws *wsi, enum lws_callback_reasons reason,
                           void *user, void *in, size_t len)
{
    switch (reason)
    {
    case LWS_CALLBACK_CLIENT_ESTABLISHED:
        {
            char clientid[10] = "client1";
            printf("Connected to server\n");
        }
        break;

    case LWS_CALLBACK_CLIENT_RECEIVE:
        // Handle received data
        printf("Received message from server: %.*s", (int)len, (char *)in);      
         const char *message = "HIIII";  
         send_ws_text_message(wsi, message);
        break;

    case LWS_CALLBACK_CLIENT_APPEND_HANDSHAKE_HEADER:
        {
            char **p = (char **)in;
            char *end = *p + len; 
            const char *auth_header = "6de1f1c-e0e9-f71659R";
            size_t auth_header_len = strlen(auth_header);

            if (lws_add_http_header_by_name(wsi, (unsigned char *)"AUTHORIZATION", (unsigned char *)auth_header, auth_header_len, (unsigned char **)p, (unsigned char *)end))
            {
                return -1;
            }
        }
        break;

    case LWS_CALLBACK_CLIENT_CONNECTION_ERROR:
        fprintf(stderr, "Connection failed\n");
        force_exit = 1;
        break;

    case LWS_CALLBACK_CLOSED:
        printf("Connection closed\n");
        force_exit = 1;
        break;

    default:
        break;
    }

    return 0;
}

static struct lws_protocols protocols[] = {
    {"http", callback_client, 0, 10},
    {NULL, NULL, 0, 0}};

int main(void)
{
    struct lws_context_creation_info info;
    struct lws_client_connect_info connect_info;
    struct lws_context *context;

    memset(&info, 0, sizeof info);
    memset(&connect_info, 0, sizeof connect_info);

    signal(SIGINT, sighandler);

    info.port = CONTEXT_PORT_NO_LISTEN;
    info.protocols = protocols;
    info.options = LWS_SERVER_OPTION_DO_SSL_GLOBAL_INIT;
    info.timeout_secs = 10; // Set timeout to 10 seconds (adjust as needed)

    context = lws_create_context(&info);
    if (!context)
    {
        fprintf(stderr, "libwebsocket init failed\n");
        return -1;
    }

    connect_info.context = context;
    connect_info.address = "172.16.100.44";
    connect_info.port = 8080;
    connect_info.path = "/";
    connect_info.host = connect_info.address;
    connect_info.origin = connect_info.address;
    connect_info.ietf_version_or_minus_one = -1;
    connect_info.protocol = protocols[0].name;

    struct lws *wsi = lws_client_connect_via_info(&connect_info);
    if (!wsi)
    {
        fprintf(stderr, "Connection failed\n");
        lws_context_destroy(context);
        return -1;
    }

    wsi_global = wsi;


     const char *message = "Hello";
    while (!force_exit)
    {
        lws_service(context, 50);

    }

    lws_context_destroy(context);

    return 0;
}
