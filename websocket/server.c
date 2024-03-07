#include <libwebsockets.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <cJSON.h>

#define USERNAME "admin"
#define PASSWORD "admin"
#define JSON_PAYLOAD_auth "{\"status\":\"OK\",\"statusCode\":0,\"payload\":{\"id\":\"53f1c-e0e9-4a7d\",\"token\":\"6de1f1c-e0e9-f71659R\"}}"
#define JSON_PAYLOAD_unauth "{\"status\":\"Unauthorized\",\"statusCode\":1,\"payload\":\"invalid\"}"
#define PAYLOAD_LEN_auth (sizeof(JSON_PAYLOAD_auth) - 1)
#define PAYLOAD_LEN_unauth (sizeof(JSON_PAYLOAD_unauth) - 1)
#define token "6de1f1c-e0e9-f71659R"

struct lws *wsi = NULL;
char *txt;
pthread_t input_thread;
struct lws **clients; // Pointer to store connected clients
pthread_mutex_t clients_mutex = PTHREAD_MUTEX_INITIALIZER;
size_t num_clients = 0;
struct per_session_data
{
    int client_id;
};

void *input_handler(void *arg)
{
    char *input_buffer = NULL;
    size_t input_buffer_size = 0;
    while (1)
    {
        // Read input line by line
        ssize_t input_length = getline(&input_buffer, &input_buffer_size, stdin);
        if (input_length == -1)
        {
            break;
        }

        int chosen_client_index;
        printf("Enter client: ");
        if (scanf("%d", &chosen_client_index) != 1)
        {
            break;
        }

        int c;
        while ((c = getchar()) != '\n' && c != EOF);

        pthread_mutex_lock(&clients_mutex);
        if (chosen_client_index >= 0 && chosen_client_index < num_clients && clients[chosen_client_index] != NULL)
        {
            lws_write(clients[chosen_client_index], input_buffer, input_length, LWS_WRITE_TEXT);
        }
        pthread_mutex_unlock(&clients_mutex);
    }

    // Clean up allocated memory
    free(input_buffer);

    return NULL;
}


static int callback_http(struct lws *wsi, enum lws_callback_reasons reason, void *user, void *in, size_t len)
{
    struct per_session_data *pss = (struct per_session_data *)user;

    switch (reason)
    {
    case LWS_CALLBACK_HTTP_BODY:
        char *body = (char *)in;

        cJSON *root = cJSON_Parse(body);
        if (root == NULL)
        {
            printf("Error parsing JSON.\n");
            return -1;
        }

        // Get username and password fields
        cJSON *username_json = cJSON_GetObjectItemCaseSensitive(root, "username");
        cJSON *password_json = cJSON_GetObjectItemCaseSensitive(root, "password");

        if (cJSON_IsString(username_json) && cJSON_IsString(password_json))
        {
            // Compare username and password
            if (strcmp(username_json->valuestring, USERNAME) == 0 && strcmp(password_json->valuestring, PASSWORD) == 0)
            {
                printf("Authentication successful.\n");
                char headers[sizeof("HTTP/1.1 200 OK\x0d\x0a"
                                    "Content-Type: application/json\x0d\x0a"
                                    "Content-Length: ") +
                             10];
                sprintf(headers, "HTTP/1.1 200 OK\x0d\x0a"
                                 "Content-Type: application/json\x0d\x0a"
                                 "Content-Length: %ld\x0d\x0a\x0d\x0a",
                        PAYLOAD_LEN_auth);

                lws_write(wsi, (unsigned char *)headers, strlen(headers), LWS_WRITE_HTTP);
                lws_write(wsi, (unsigned char *)JSON_PAYLOAD_auth, PAYLOAD_LEN_auth, LWS_WRITE_HTTP_FINAL);

                lws_return_http_status(wsi, HTTP_STATUS_OK, NULL);
                lws_get_close_payload(wsi);
            }

            else
            {
                printf("Authentication failed.\n");
                char headers[sizeof("HTTP/1.1 401 UNAUTHORIZED\x0d\x0a"
                                    "Content-Type: application/json\x0d\x0a"
                                    "Content-Length: ") +
                             10];
                sprintf(headers, "HTTP/1.1 401 UNAUTHORIZED\x0d\x0a"
                                 "Content-Type: application/json\x0d\x0a"
                                 "Content-Length: %ld\x0d\x0a\x0d\x0a",
                        PAYLOAD_LEN_unauth);

                lws_write(wsi, (unsigned char *)headers, strlen(headers), LWS_WRITE_HTTP);
                lws_write(wsi, (unsigned char *)JSON_PAYLOAD_unauth, PAYLOAD_LEN_unauth, LWS_WRITE_HTTP_FINAL);

                lws_return_http_status(wsi, HTTP_STATUS_UNAUTHORIZED, NULL);
                lws_get_close_payload(wsi);
            }
        }
        else
        {
            printf("Invalid JSON format.\n");
                            return -1;
        }

        cJSON_Delete(root);
        break;

    case LWS_CALLBACK_FILTER_PROTOCOL_CONNECTION:
    {
        char auth[1024];
        int auth_len;
        auth_len = lws_hdr_copy(wsi, auth, sizeof(auth), WSI_TOKEN_HTTP_AUTHORIZATION);
        if (auth_len > 0)
        {
            auth[auth_len] = '\0';
        }

        if (auth_len > 0)
        {
            printf("Token: %s\n", token);
            if (strcmp(token, auth) == 0)
            {
                printf("Token matched. Allowing connection upgrade.\n");
                return 0;
            }
            else
            {
                printf("Invalid Token. Denying connection upgrade.\n");
                return 1;
            }
        }
        else
        {
            printf("Token not found. Denying connection upgrade.\n");
            return 1;
        }
        break;
    }

    case LWS_CALLBACK_ESTABLISHED:
        printf("Client connected.\n");
        // Find an available slot in the clients array and store the client
            pthread_mutex_lock(&clients_mutex);

            // Resize the clients array
            clients = realloc(clients, (num_clients + 1) * sizeof(struct lws *));
            if (!clients)
            {
                // Handle realloc failure
                printf("Failed to resize clients array.\n");
                pthread_mutex_unlock(&clients_mutex);
                return -1;
            }

            clients[num_clients] = wsi;
            pss->client_id = num_clients; // Assign a unique client ID

            num_clients++;

            pthread_mutex_unlock(&clients_mutex);
            printf("client Index: %d\n", pss->client_id);
            lws_callback_on_writable(wsi);
        break;

    case LWS_CALLBACK_CLIENT_HTTP_WRITEABLE:        
            lws_write(wsi, txt, strlen(txt), LWS_WRITE_TEXT);
        break;

    case LWS_CALLBACK_RECEIVE:
        printf("Received: %.*s from client %d\n", (int)len, (char *)in, pss->client_id);
        break;

    case LWS_CALLBACK_CLOSED:
        printf("Client disconnected.\n");
        pthread_mutex_lock(&clients_mutex);
        if (pss->client_id >= 0 && pss->client_id < num_clients)
        {
            clients[pss->client_id] = NULL;

            for (size_t i = pss->client_id; i < num_clients - 1; ++i)
            {
                clients[i] = clients[i + 1];
            }

            num_clients--;

            if (num_clients > 0)
            {
                struct lws **temp_clients = realloc(clients, num_clients * sizeof(struct lws *));
                if (temp_clients)
                {
                    clients = temp_clients;
                }
                else
                {
                    printf("Failed to resize clients array.\n");
                }
            }
            else
            {
                free(clients);
                clients = NULL;
            }
        }
        pthread_mutex_unlock(&clients_mutex);
        break;

    default:
        break;
    }

    return 0;
}

int main(int argc, char **argv)
{
    // Allocate dynamic memory for txt
    txt = (char *)malloc(20 * sizeof(char));

    if (!txt)
    {
        printf("Failed to allocate memory for txt.\n");
        return -1;
    }

    static struct lws_protocols protocols[] = {
        {"http",
         callback_http,
         sizeof(struct per_session_data),
         0,
         0,
         0,
         0}, 
        {NULL, NULL, 0, 0, 0, 0, 0}};

    struct lws_context_creation_info info;
    const char *iface = "your_IP_addr"; //add your IP address
    int port = 8080;
    int opts = 0;

    memset(&info, 0, sizeof(info));
    info.port = port;
    info.iface = iface;
    info.protocols = protocols;
    info.extensions = NULL;
    info.gid = -1;
    info.uid = -1;

    struct lws_context *context = lws_create_context(&info);
    pthread_create(&input_thread, NULL, input_handler, NULL);

    if (!context)
    {
        printf("Failed to create WebSocket context.\n");
        free(txt);
        return -1;
    }

    while (1)
    {
        lws_service(context, 50);
    }

    pthread_join(input_thread, NULL);
    lws_context_destroy(context);
    free(txt);

    return 0;
}
