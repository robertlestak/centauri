# Centauri - decentralized p2p data bus

[Read the Docs](https://centauri.sh/docs/)

Centauri is a decentralized, peer-to-peer (p2p), asynchronous, ephemeral, and fault-tolerant data bus. Centauri enables disconnected users around the world to transfer files, messages, and any other type of data without having a direct network connection between nodes or concurrent active sessions for p2p transfers.

Centauri is similar to cryptocurrencies such as Ethereum and Bitcoin in that it utilizes a globally replicated network to exchange information secured with cryptographic keys, however unlike conventional cryptocurrency technologies, instead of blockchain linked lists, Centauri more closely mirrors Hyperledger Fabric's network model of peers communicating over gossip protocol to share current message state, with a hint of bittorrent's p2p file transfer model.
 
## The Centauri Network

Modern cryptocurrencies such as Bitcoin and Ethereum rely on a single global network unifed through a singular blockchain as tracked from the genesis block. State added to _any_ node (assuming the block has been accepted) will be immutably added to the blockchain database on every _other_ node in the network, ostensibly for all eternity from that point forward.

This guarantees the state on the chain is provably accurate, enabling many of the innovations we have seen in the cryptocurrency space. However this immutability and global replication means that the blockchain network is inefficient for larger data transfers / storage.

If a user in Bangladesh needs to send 50Mb of data to a user in Brazil, they could add this data to a cryptocurrency transaction and the recpient can then read the data off the transaction when their local node processes the block, but this now means every other node in the network will forever have those 50Mb consumed, both in terms of chain usage, as well as node storage.

In a Centauri network, the sending user can upload the 50Mb data (encrypted with the recipient's public key) to their local Centauri peer, which will then replicate this data to all of the connected Centauri peers. When the recieveing user connects to their local Centauri peer, they can retrieve the data locally. Once the receiving user has confirmed receipt, the data is removed from all connected Centauri peers.

This ephemerality of data in-network enables Centauri to support larger message throughputs without significantly increasing the resource requirements of Centauri peer nodes.

A Centauri network can be launched in an entirely isolated RFC1918 network, with no connection to the WAN at all, and the Centauri peers and nodes within this network will continue to operate without issue. Similarly, a Centauri peer can advertise a NAT'ed address to enable "extranet" access to a closed Centauri network.