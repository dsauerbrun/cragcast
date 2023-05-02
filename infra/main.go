package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	nixosAMIOwner    = "080433136561"
	nixOSImageName   = "NixOS-22.11.466.*-aarch64-linux"
	cidr             = "192.168.1.0/24"
	instanceType     = "a1.large"
	defaultTagName   = "cragcast"
	keyName          = "cragcast"
	allCIDRBlock     = "0.0.0.0/0"
	availabilityZone = "us-west-2a"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ami, err := resolveArmNixAMI(ctx)
		if err != nil {
			return err
		}
		fmt.Println(ami)

		vpc, err := createVPC(ctx)
		if err != nil {
			return err
		}

		igw, err := createInternetGateway(ctx, vpc.ID())
		if err != nil {
			return err
		}

		rt, err := createRouteTables(ctx, vpc.ID(), igw.ID())
		if err != nil {
			return err
		}

		subnet, err := createSubnet(ctx, vpc.ID(), rt.ID())
		if err != nil {
			return err
		}

		sg, err := createSecurityGroup(ctx, vpc.ID())
		if err != nil {
			return err
		}

		err = createInstance(ctx, ami, subnet.ID(), sg.ID())
		if err != nil {
			return err
		}

		return nil
	})
}

// createVPC makes a new VPC with all default settings and the CIDR set to the const [cidr].
func createVPC(ctx *pulumi.Context) (*ec2.Vpc, error) {
	args := &ec2.VpcArgs{
		Tags:      pulumi.ToStringMap(map[string]string{"Name": defaultTagName}),
		CidrBlock: pulumi.StringPtr(cidr),
	}
	vpc, err := ec2.NewVpc(ctx, "cragcast", args)
	if err != nil {
		return nil, err
	}
	return vpc, nil
}

// createInternetGateway makes a new gateway where external traffic can be routed through.
func createInternetGateway(ctx *pulumi.Context, vpcID pulumi.IDOutput) (*ec2.InternetGateway, error) {
	args := &ec2.InternetGatewayArgs{
		Tags:    pulumi.ToStringMap(map[string]string{"Name": defaultTagName}),
		TagsAll: nil,
		VpcId:   vpcID,
	}
	igw, err := ec2.NewInternetGateway(ctx, "cragcast", args)
	if err != nil {
		return nil, err
	}
	return igw, nil
}

// createSubnet makes a new subnet that takes up the entire space of [cidr].
func createSubnet(ctx *pulumi.Context, vpcID pulumi.IDOutput, rtID pulumi.IDOutput) (*ec2.Subnet, error) {
	args := &ec2.SubnetArgs{
		AvailabilityZone: pulumi.StringPtr(availabilityZone),
		CidrBlock:        pulumi.StringPtr(cidr),
		// TODO(joshrosso): we can append a set of default tags to the provider.
		//		    should look into this
		Tags:  pulumi.ToStringMap(map[string]string{"Name": defaultTagName}),
		VpcId: vpcID,
	}
	subnet, err := ec2.NewSubnet(ctx, "cragcast", args)
	if err != nil {
		return nil, err
	}
	ec2.NewRouteTableAssociation(ctx, "rt-association", &ec2.RouteTableAssociationArgs{
		RouteTableId: rtID,
		SubnetId:     subnet.ID(),
	})
	return subnet, nil
}

// createRouteTables configures routes such that all public traffic (0.0.0.0/0)
// goes through the internet gateway and continues to allow all private traffic
// ([cidr]) to move through the VPC.
func createRouteTables(ctx *pulumi.Context, vpcID pulumi.IDOutput, igwID pulumi.IDOutput) (*ec2.RouteTable, error) {
	// Create a new private route table and associate it with the VPC
	rt, err := ec2.NewRouteTable(ctx, "private-route-table", &ec2.RouteTableArgs{
		VpcId: vpcID,
	})
	if err != nil {
		return nil, err
	}

	// Add a route to the public route table that routes all traffic through the internet gateway
	_, err = ec2.NewRoute(ctx, "public-route", &ec2.RouteArgs{
		RouteTableId:         rt.ID(),
		DestinationCidrBlock: pulumi.String(allCIDRBlock),
		GatewayId:            igwID,
	})
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// resolveArmNixAMI looks up the AMI ID for the AMI name specified in [nixOSImageName].
func resolveArmNixAMI(ctx *pulumi.Context) (string, error) {
	amiFilter := ec2.GetAmiIdsArgs{
		Filters: []ec2.GetAmiIdsFilter{
			{
				Name:   "name",
				Values: []string{nixOSImageName},
			},
		},
		Owners: []string{nixosAMIOwner},
	}
	amiIDs, err := ec2.GetAmiIds(ctx, &amiFilter)
	if err != nil {
		return "", fmt.Errorf("failed retrieving AMI Ids for %s. Error: %s", nixOSImageName, err)
	}
	// If more than 1 AMI is returned, we should error. We shouldn't rely
	// on "latest" as that can give us inconsistent results between builds.
	// While this should never be > 1, if it is, we need to consider
	// adjustments to our filters.
	if len(amiIDs.Ids) > 1 {
		return "", fmt.Errorf("resolved more than 1 AMI for name %s. Should be exactly 1. Look to further constrain the nixOSImageName.", nixOSImageName)
	}
	// this is a safe operation *only* because of the length check above.
	// If that's removed, this should be adjusted.
	id := amiIDs.Ids[0]
	args := ec2.LookupAmiArgs{
		Filters: []ec2.GetAmiFilter{
			{
				Name:   "image-id",
				Values: []string{id},
			},
		},
	}
	amiDetails, err := ec2.LookupAmi(ctx, &args)
	if err != nil {
		panic(err)
	}
	log.Printf("AMI %s resolved:\n %+v\n", id, amiDetails)
	return id, nil
}

// createSecurityGroup makes a new security groups allowing all egress traffic and limiting ingress traffic to SSH and :80/:443.
func createSecurityGroup(ctx *pulumi.Context, vpcID pulumi.IDOutput) (*ec2.SecurityGroup, error) {
	sg, err := ec2.NewSecurityGroup(ctx, "my-security-group", &ec2.SecurityGroupArgs{
		VpcId: vpcID,
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String(allCIDRBlock)},
			},
		},
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(22),
				CidrBlocks: pulumi.StringArray{pulumi.String(allCIDRBlock)},
			},
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(443),
				CidrBlocks: pulumi.StringArray{pulumi.String(allCIDRBlock)},
			},
			// TODO(joshrosso): let's close this one up once we have all the SSL stuff automated.
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{pulumi.String(allCIDRBlock)},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return sg, nil
}

// createInstance makes a new EC2 instance (VM) inside of the specified subnet.
func createInstance(ctx *pulumi.Context, amiID string, subnetID pulumi.IDOutput, sgIDs pulumi.IDOutput) error {
	instance, err := ec2.NewInstance(ctx, "cragcast", &ec2.InstanceArgs{
		Ami:                      pulumi.StringPtr(amiID),
		AssociatePublicIpAddress: pulumi.BoolPtr(true),
		InstanceType:             pulumi.StringPtr(instanceType),
		SubnetId:                 subnetID,
		Tags:                     pulumi.ToStringMap(map[string]string{"Name": defaultTagName}),
		VpcSecurityGroupIds:      pulumi.StringArray{sgIDs},
		KeyName:                  pulumi.StringPtr(keyName),
	})
	if err != nil {
		return err
	}
	ctx.Export("public-ip", instance.PublicIp)
	return nil
}
