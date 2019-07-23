##Current version 1.8.27-atlantis

## Go GUBET

The GUBET protocol is implemented in Golang and uses same Ethash algorithm as Ethereum blockchain.


Building UbetscoinGo client requires both a Go (version 1.8 or later) and a C compiler.

## Build instructions for Mac

Clone the repository to a directory of your choosing:
```
git clone https://github.com/ubetprotocol/go-ubetprotocol
```

Building UbetscoinGo requires the Go compiler:
```
brew install go
```
Finally, build the UbetscoinGo program using the following command.
```
cd go-ubetprotocol
make gubet
```
If you see some errors related to header files of Mac OS system library, install XCode Command Line Tools, and try again.
```
xcode-select --install
```
You can now run ```build/bin/gubet``` to start your node. 

You can also run ```sudo cp ./build/bin/gubet /usr/local/bin/gubet``` and then run ```gubet``` from any directory.   

go-ubetprotocol client will now start downloading blocks. Wait till your node synchronises with the network to start activities such as mining.
## Build instructions for Windows

The Chocolatey package manager provides an easy way to get the required build tools installed. If you don't have chocolatey yet, follow the instructions on https://chocolatey.org to install it first.

Then open an Administrator command prompt and install the build tools we need:
```
C:\Windows\system32> choco install git
C:\Windows\system32> choco install golang
C:\Windows\system32> choco install mingw
```

Installing these packages will set up the Path environment variable. Open a new command prompt to get the new Path. The following steps don't need Administrator privileges.

Please ensure that the installed Go version is 1.8 (or any later version).

First we'll create and set up a Go workspace directory layout, then clone the source.

OBS If, during the commands below, you get the following message:
```
 WARNING: The data being saved is truncated to 1024 characters.
```
Then that means that the ```setx``` command will fail, and proceeding will truncate the ```Path/GOPATH```. If this happens, it's better to abort, and try to make some more room in ```Path``` before trying again.

```
C:\Users\xxx> set "GOPATH=%USERPROFILE%"
C:\Users\xxx> set "Path=%USERPROFILE%\bin;%Path%"
C:\Users\xxx> setx GOPATH "%GOPATH%"
C:\Users\xxx> setx Path "%Path%"
C:\Users\xxx> mkdir src\github.com\Ubetscoin
C:\Users\xxx> git clone https://github.com/ubetprotocol/go-ubetprotocol
C:\Users\xxx> cd go-ubetprotocol
C:\Users\xxx> go get -u -v golang.org/x/net/context
```

Finally, the command to compile Ubetgo-ubetprotocol is:

```
C:\Users\xxx\src\github.com\ubetprotocol/go-ubetprotocol> go install -v ./cmd/go-ubetprotocol
```
You can now run ```build/bin/gubet.exe``` to start your node.

go-ubetprotocol client will now start downloading blocks. Wait till your node synchronises with the network to start activities such as mining.


## Build instructions for Ubuntu

Clone the repository to a directory of your choosing:
```
git clone https://github.com/ubetprotocol/go-ubetprotocol
```

Install latest distribution of Go (v1.8) if you don't have it already. 

Building ```ubetprotocol``` requires Go and C compilers to be installed:
```
sudo apt-get install -y build-essential golang
```
Finally, build the ```ubetprotocol``` program using the following command.
```
cd go-ubetprotocol
make gubet
```
You can now run ```build/bin/gubet``` to start your node. 

You can also run ```sudo cp ./build/bin/gubet /usr/local/bin/gubet``` and then run ```gubet``` from any directory. 

ubetprotocol client will now start downloading blocks. Wait till your node synchronises with the network to start activities such as mining.


### Programatically interfacing UbetscoinGo nodes

ubetprotocol has built in support for a JSON-RPC based APIs. These can be
exposed via HTTP, WebSockets and IPC (unix sockets on unix based platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by ubetprotocol, whereas the HTTP
and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons.
These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpcport` HTTP-RPC server listening port (default: 8545)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: "localhost")
  * `--wsport` WS-RPC server listening port (default: 8546)
  * `--wsapi` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect
via HTTP, WS or IPC to a ubetprotocol node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification)
on all transports. You can reuse the same connection for multiple requests!


## License

The go-ubetprotocol library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-GUBET binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.