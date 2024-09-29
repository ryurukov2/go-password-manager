This is a simple password manager CLI tool.


## Usage
- download the executable (file named password-manager from the github repo)
- run the program
- available commands and flags after authentication with master password:
    -- add -s=<service> -u=<username> -p=<password>
    -- get -s=<service>
    -- delete -s=<service> -u=<username>
    -- exit


## Features:
- master password that is never stored anywhere
    -- set up master password on the first run
    -- secure password field that doesn't expose the master password while it is being entered (similar to how linux handles sudo password in terminal)
    -- enter master password on subsequent runs to be able to decrypt other saved passwords
- store usernames and passwords for any service/website
    -- usernames and passwords are encrypted before being saved and decrypted when asked for, using a key derived from the master password
- delete any saved service/username/password combination
