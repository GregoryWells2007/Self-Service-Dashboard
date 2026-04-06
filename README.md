# FreeIPA Account Manager

A simple, lightweight web application for managing user profile photos in a FreeIPA server.

## Features
*   **LDAP Authentication**: Secure login with existing FreeIPA credentials.
*   **Profile Management**: View user details (Display Name, Email).
*   **Photo Upload**: Users can upload and update their profile picture (`jpegPhoto` attribute).
*   **Change Password**: Users can change there password once logged in
*   **Session Management**: Secure, cookie-based sessions with CSRF protection.
*   **Customizable**: Configurable styling (logo, favicon) and LDAP settings.

## Prerequisites
*   **Go 1.26+** installed on your machine.
*   Access to a **FreeIPA Server**.
*   A Service Account with permission to search and modify the `jpegPhoto` attribute.

## Setup & Installation

1.  **Clone the Repository**
    ```bash
    git clone https://git.astraltech.xyz/gawells/Self-Service-Dashboard.git
    cd Self-Service-Dashboard
    ```

2.  **Configure the Application**
    Copy the example configuration file to the production path:
    ```bash
    cp data/config.example.json data/config.json
    ```

5. **Edit config**
    Edit the config file
    ```bash
    nvim data/config.json
    ``` 

4.  **Install Dependencies**
    ```bash
    go mod tidy
    ```

5.  **Run the Server**
    ```bash
    go run ./src/main/
    ```
    The application will be available at `http://localhost:<port>`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
