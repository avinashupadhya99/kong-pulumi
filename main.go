package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")
		key_name := conf.Require("key_name")
		public_key := conf.Require("public_key")
		ec2_ami := conf.Require("ec2_ami")

		kong_vpc, err := ec2.NewVpc(ctx, "kong-network", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
		})
		if err != nil {
			return err
		}
		kong_subnet, err := ec2.NewSubnet(ctx, "kong-gateway-subnetwork", &ec2.SubnetArgs{
			VpcId:     kong_vpc.ID(),
			CidrBlock: pulumi.String("10.0.1.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Kong"),
			},
		})
		if err != nil {
			return err
		}

		internet_gateway, err := ec2.NewInternetGateway(ctx, "gw", &ec2.InternetGatewayArgs{
			VpcId: kong_vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Kong"),
			},
		})
		if err != nil {
			return err
		}

		route_table, err := ec2.NewRouteTable(ctx, "kong-route-table", &ec2.RouteTableArgs{
			VpcId: kong_vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: internet_gateway.ID(),
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Kong"),
			},
		})
		if err != nil {
			return err
		}

		_, errr := ec2.NewMainRouteTableAssociation(ctx, "mainRouteTableAssociation", &ec2.MainRouteTableAssociationArgs{
			VpcId:        kong_vpc.ID(),
			RouteTableId: route_table.ID(),
		})
		
		
		// subnetId := kong_subnet.ID().ApplyT(func(id string) string {
		// 	return id
		// }).(pulumi.StringOutput)

		// route_table, err := ec2.LookupRouteTable(ctx, &ec2.LookupRouteTableArgs{
		// 	SubnetId: pulumi.String(subnetId),
		// }, nil)
		// if err != nil {
		// 	return err
		// }

		// _, errr := ec2.NewRoute(ctx, "internet-route", &ec2.RouteArgs{
		// 	RouteTableId:           pulumi.String(route_table.Id),
		// 	DestinationCidrBlock:   pulumi.String("10.0.1.0/24"),
		// 	GatewayId:				internet_gateway.ID(),
		// })
		if errr != nil {
			return errr
		}

		key_pair, err := ec2.NewKeyPair(ctx, key_name, &ec2.KeyPairArgs{
			KeyName: pulumi.String(key_name),
			PublicKey: pulumi.String(public_key),
		})
		if err != nil {
			return err
		}

		// Create a new security group for port 8000 and 8001.
		group, err := ec2.NewSecurityGroup(ctx, "kong-secgrp", &ec2.SecurityGroupArgs{
			VpcId: kong_vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(8000),
					ToPort:     pulumi.Int(8000),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(8002),
					ToPort:     pulumi.Int(8002),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
					Ipv6CidrBlocks: pulumi.StringArray{pulumi.String("::/0")},
				},
			},
		})
		if err != nil {
			return err
		}

		gateway, err := ec2.NewInstance(ctx, "kong-gateway", &ec2.InstanceArgs{
			Tags:                pulumi.StringMap{"Name": pulumi.String("kong-gateway")},
			InstanceType:        pulumi.String("t2.micro"), // t2.micro is available in the AWS free tier.
			VpcSecurityGroupIds: pulumi.StringArray{group.ID()},
			AssociatePublicIpAddress: pulumi.Bool(true),
			Ami:                 pulumi.String(ec2_ami),
			KeyName:			 pulumi.String(key_name),
			SubnetId:			 kong_subnet.ID(),
			UserData: pulumi.String(`#!/bin/bash
echo "deb [trusted=yes] https://download.konghq.com/gateway-3.x-ubuntu-$(lsb_release -sc)/ default all" | sudo tee /etc/apt/sources.list.d/kong.list
sudo apt-get update
sudo apt install -y kong-enterprise-edition=3.0.1.0 jq
sudo apt update
sudo apt install postgresql postgresql-contrib -y
sudo systemctl start postgresql.service
sudo curl -o /etc/kong/kong.conf https://gist.githubusercontent.com/avinashupadhya99/384cbf15a73060a8347982cd4b6a6ddc/raw/87453d6b78b042b472ca855b2278c0fb2a0a08c5/kong.conf
sudo -u postgres psql -c "CREATE USER kong WITH PASSWORD 'super_secret';"
sudo -u postgres psql -c "CREATE DATABASE kong OWNER kong;"
sudo kong migrations bootstrap
sudo kong start`),
		},
		pulumi.DependsOn([]pulumi.Resource{key_pair}))

		// Export the resulting server's IP address and DNS name.
		ctx.Export("publicIp", gateway.PublicIp)
		ctx.Export("publicHostName", gateway.PublicDns)
		return nil
	})
}