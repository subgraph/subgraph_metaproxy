# Subgraph Metaproxy

Subgraph Metaproxy is a proxy redirector. It transparently redirects (with the
aid of netfilter/iptables) communications to a proxy server (such as SOCKS5
or HTTP). 

Metaproxy should run on a variety of Linux-based operating systems but it has
only been tested on Debian and Subgraph OS. 

It does depend on a Linux kernel with NAT + redirect capabilities and iptable
rules to redirect outgoing traffic to Metaproxy.

Metaproxy is inspired by the [redsocks](http://darkk.net.ru/redsocks/) project 
(and has similar functions at this stage).

## Use cases

Subgraph OS uses Metaproxy to redirect non-proxy aware applications through Tor.
This is backed by a set of iptables rules that block other outgoing connections
that are not destined for the Tor network. This is known as Tor enforcement.

But generally, Metaproxy can be used to redirect specific connections through a
SOCKS or HTTP proxy (such as privoxy or polipo).

## Usage

### Running Subgraph Metaproxy

To run Metaproxy, some iptables rules must first be created.

1. Create a new chain for METAPROXY

```
$ iptables -t nat -N METAPROXY
```

2. Create rules to skip processing of destination of reserved local networks

```
# iptables -t nat -A METAPROXY -d 0.0.0.0/8 -j RETURN 

# iptables -t nat -A METAPROXY -d 10.0.0.0/8 -j RETURN

# iptables -t nat -A METAPROXY -d 127.0.0.0/8 -j RETURN

# iptables -t nat -A METAPROXY -d 169.254.0.0/16 -j RETURN

# iptables -t nat -A METAPROXY -d 172.16.0.0/12 -j RETURN
 
# iptables -t nat -A METAPROXY -d 192.168.0.0/16 -j RETURN                                                                                                                                   
# iptables -t nat -A METAPROXY -d 224.0.0.0/4 -j RETURN

# iptables -t nat -A METAPROXY -d 240.0.0.0/4 -j RETURN   
```

3. Create the rule to redirect traffic to the default port of Metaproxy 
(8675/tcp)


```
# iptables -t nat -A METAPROXY -p tcp -j REDIRECT --to-ports 8675                                                                                                                           
```

4. Start the Metaproxy with the provided config file:

```
$ subgraph_metaproxy -c subgraph_metaproxy.conf
```

### Testing the config file

Metaproxy provides a command-line option to test the config file. It is a good
idea to run this as a sanity check prior to starting the Metaproxy, especially
if you have made changes.

```
$ subgraph_metaproxy -c subgraph_metaproxy.conf -t
```

## Caveats

1. Subgraph Metaproxy is currently alpha quality software -- it may not be that
efficient or reliable in its current stage of development.

2. Do not run Metaproxy in debug mode if your value your anonymity as debug
does not sanitize connection details such as the destination IP of connections.

## Limitations

1. Subgraph Metaproxy requires iptables rules that to redirect traffic to its
listening port.

2. By itself, Metaproxy will not block DNS leaks. A Tor enforcement policy
at the netfilter/iptables level is better suited to addressing this issue.

3. Metaproxy is not going to work very well with IPv6. Tor currently doesn't 
support it either.

