# FreeIPA Account Manager

A simple, lightweight web application for managing user profile photos in a FreeIPA server.

## Features
*   **LDAP Authentication**: Secure login with existing FreeIPA credentials.
*   **Profile Management**: View user details (Display Name, Email).
*   **Photo Upload**: Users can upload and update their profile picture (`jpegPhoto` attribute).
*   **Session Management**: Secure, cookie-based sessions with CSRF protection.
*   **Customizable**: Configurable styling (logo, favicon) and LDAP settings.

## Prerequisites

*   **Go 1.20+** installed on your machine.
*   Access to an **FreeIPA Server**.
*   A Service Account (Bind DN) with permission to search users and modify the `jpegPhoto` attribute.

## Setup & Installation

1.  **Clone the Repository**
    ```bash
    git clone https://git.astraltech.xyz/gawells/Self-Service-Dashboard
    cd Self-Service-Dashboard
    ```

2.  **Configure the Application**
    Copy the example configuration file to the production path:
    ```bash
    cp data/config.example.json data/config.json
    ```

5. **Edit config**
    put in your config values for ldap, and whatevery styling guidelines you would want to use

4.  **Install Dependencies**
    ```bash
    go mod tidy
    ```

5.  **Run the Server**
    ```bash
    go run ./src/main/
    ```
    The application will be available at `http://<host>:<port>`.

## Directory Structure

*   `src/`: Go source code (`main.go`, `ldap.go`, `session.go`, etc.).
*   `src/pages/`: HTML templates for login and profile pages.
*   `static/`: CSS files, images, and other static assets.
*   `data/`: Configuration files and local assets (logos).
*   `avatars/`: Stores cached user profile photos.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
