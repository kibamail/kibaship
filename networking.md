```yaml
kubelet:
  nodeIP:
    validSubnets:
      - 10.0.1.0/24 # check -> from vswitch subnet (the node's private ip subnet. all bare metal servers are part of this subnet)
network:
  hostname: cp-1
  interfaces:
    - interface: enp5s0 # check
      addresses:
        - 65.109.58.113/26 # check -> from network discovery
      routes:
        - network: 0.0.0.0/0 # check -> fixed
          gateway: 65.109.58.65 # check -> from network discovery
      vlans:
        - vlanId: 4000 # check
          addresses:
            - 10.0.1.10/24 # check
          mtu: 1400 # check -> fixed
          routes:
            - network: 10.0.0.0/16 # check -> from networkSubnet (the whole cloud network as instructed by hetzner)
              gateway: 10.0.1.1 # check -> the first ip in the vswitch subnet
```
