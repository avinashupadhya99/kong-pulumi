# Kong Pulumi

Pulumi program in Go to create EC2 and necessary infra in AWS to deploy Kong. 
The program deploys Kong on the EC2 instance as well.


## Pulumi config variables

```
pulumi config set ec2_ami <ami>
pulumi config set key_name <key_name>
pulumi config set public_key <public_key_for_ssh>
```
