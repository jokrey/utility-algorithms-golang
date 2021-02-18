# utility-algorithms-golang

My very own, multi purpose utility algorithms.
The algorithms and classes are all cool, but each not big enough for their own repository.

Some algorithms exist in a similar form in other repositories

Use this project by adding the following in your go.mod:

```
require (
  github.com/jokrey/utility-algorithms-golang v1.0.6
)
```


## wsclientable

(apologies for the package)

Interoperable with the wsclientable client in the utility-algorithms-flutter package.

Based on WebSockets
  * adds the concept of typed messages
    * messages have a 'type'
    * types can be subscribed to on clients
    * clients now have multiple, separated streams of inputs
  * adds the concept of forwarding
    * connections have an id
    * connections can be stored
    * connections can be adressed by their id
    * connections can send each other messages
  * adds the concept of forwarding within rooms
    * rooms separate the server into distinct sections
    * to the client it looks and feels like its on different servers
    * rooms can allow only certain clients
    * rooms can be permanent
    * rooms can be temporary and self deleting
    * rooms can be repeating (closing connections when they become invalid)
    * rooms can be edited over local http requests

## webrtc (signaling)

Uses wsclientable to support message passing between webrtc connections
(by simply supporting forwarding for the messageTypes "offer", "answer", "candidate").

Out-of-the-box interoperable with the utility-algorithms-flutter webrtc support.


## mcnp

Multi Chunk Network Protocol

Adds very little functionality to tcp.
Namely, re-adds packets(chunks), by sending the number of upcoming bytes in the chunk every time.

Further adds the very simple concept of 'causes' to allow distinguishing message types(causes) out of order.

Additionally, supports a couple of data types in multiple programming languages for integrated interoperability.


## stringencoder

Tagged strings to string conversion. Supports arrays, nesting and a number of common data types.

Essentially a poor man json,
without simple human mutability, worse readability and better performance only in some use cases.

I like it still, but I recognise my bias.