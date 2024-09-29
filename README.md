This is a simple password manager CLI tool.


## Usage
- download the right executable for your OS from the [releases page](https://github.com/ryurukov2/go-password-manager/releases/latest), or if you want to compile it yourself - scroll down to the Compile instructions
- if you download the executable, it is recommended to move it out of your Downloads folder into a folder of its own (e.g. C:/Program Files/Password Manager)
- run the program
- available commands and flags after authentication with master password:
    -- add -s=service name -u=username -p=password
    -- get -s=service name
    -- delete -s=service name -u=username
    -- exit
    (!) there should be no whitespaces in the service name, username or password


## Features:
- master password that is never stored anywhere
    -- set up master password on the first run
    -- secure password field that doesn't expose the master password while it is being entered (similar to how linux handles sudo password in terminal)
    -- enter master password on subsequent runs to be able to decrypt other saved passwords
- store usernames and passwords for any service/website
    -- usernames and passwords are encrypted before being saved and decrypted when asked for, using a key derived from the master password
- delete any saved service/username/password combination

## Potential issues and fixes

- on Linux, when downloading the compiled executable, often it will not have execute permissions by default. To fix this, navigate to the directory where the file is and run **chmod +x password-manager-linux-amd64** in the terminal

## Compile instructions
- install [Go](https://go.dev/doc/install)
- clone the repository into a folder
- run **go build**. This will compile the program with the name of the folder. Optionally, you can run **go build -o name of your choice**