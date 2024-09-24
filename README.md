# Gyazo Uploader

This is a simple script to upload images to private Gyazo server.

## Installation

### 1. Install Flameshot

Flameshot is a powerful yet simple-to-use screenshot software. You can install it using the following command:

```shell
# Ubuntu
apt install flameshot

# Fedora
dnf install flameshot
```

Config Flameshot Save Path to specific folder, e.g., /tmp/screenshots. Take a note of this path as you will need it later.

Next config Flameshot to check "Save image after copy" to save the image to the specified folder.

### 2. Set the shortcut to run Flameshot
For Ubuntu, you can set the shortcut by going to Settings -> Keyboard Shortcuts -> Custom Shortcuts -> Add custom shortcut. Set the command to run Flameshot. For example, I set the command `Ctrl + Shift + F` to run Flameshot to `flameshot gui`.

### 3. Build the Gyazo Uploader

First, ensure you have Go installed on your system. Then, clone the repository and build the program:
```shell
git clone https://github.com/tdkhoavn/gyazo-uploader.git
cd gyazo-uploader
go build -o gyazo-uploader main.go
```
Second, edit your .gyazo.config.yml file to include your Gyazo API key and the path to the folder where Flameshot saves the images. For example:
```yaml
host: # Your Gyazo server
cgi: # Your Gyazo upload CGI path
http_port: 443
use_ssl: yes
mark_important: yes
watch_dir: /tmp/screenshots # Path to the folder where Flameshot saves the images
```
### 4. Set up the Gyazo Uploader
Copy gyazo-uploader binary file to a folder `/urs/local/bin/`.
```shell
sudo cp gyazo-uploader /usr/local/bin/
```

To set up systemd service to run the Gyazo Uploader script. Create a file `/etc/systemd/system/gyazo-uploader.service` with the following content:
```shell
[Unit]
Description=Gyazo Uploader
After=network.target

[Service]
ExecStart=/usr/local/bin/gyazo-uploader
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

Reload systemd to recognize the new service:
```shell
systemctl daemon-reload
```

Enable the service to start on boot:  
```shell
sudo systemctl enable gyazo-uploader.service
```

Start the service:
```shell
sudo systemctl start gyazo-uploader.service
```

Check the status of the service:
```shell
sudo systemctl status gyazo-uploader.service
```



## Usage
Take a screenshot using Flameshot with shortcut key.
The image will be uploaded to Gyazo, and the URL will be opened in your default browser.