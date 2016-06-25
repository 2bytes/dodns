# DNS updater for Digital Ocean
This tool is in no way affiliated with Digital Ocean, I just happen to use Digital Ocean and have decided that it is useful for me to be able to update my DO hostnames programatically.

## Purpose
The purpose of this tool is to update a specific hostname in DO from the command line, if you cron this tool, you can potentially use DO like a dynamic DNS provider.

There is of course no reason you can't juse use this tool to update other A records from the command line, but that was not its intended purpose.

## Usage

Obtain a 64 digit token from your Digital Ocean account in order to use the tool, it communicates with the DO API v2.  
Call the tool from the command line as follows;
```
./dodns [-ip <ip>] <token> <domain> <record>
```

The tool will use one of the external ip search urls in the code, optionally you can provide the IP yourself from elsewhere using the `ip` flag.

## External IP retrieval
I run my own IP echo on a machine in my DO account. Three lines of code are needed to do so (see comment in code) but I would also recommend hosting it with TLS, host all your web with TLS, no excuses, Let's Encrypt is free and simple to use. If you do run your own, simply replace on or both of the URLs in the code with your own and it will use them.
