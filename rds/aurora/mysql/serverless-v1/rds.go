package main

import (
	"strings"

	"github.com/acorn-io/aws/rds"
	"github.com/acorn-io/services/aws/libs/common"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsrds"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/sirupsen/logrus"
)

var engine = awsrds.DatabaseClusterEngine_AURORA_MYSQL()

func NewRDSStack(scope constructs.Construct, props *rds.RDSStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, jsii.String("Stack"), &sprops)

	vpc := awsec2.Vpc_FromLookup(stack, jsii.String("VPC"), &awsec2.VpcLookupOptions{
		VpcId: jsii.String(props.VpcID),
	})

	subnetGroup := rds.GetPrivateSubnetGroup(stack, jsii.String("SubnetGroup"), vpc)
	sgs := &[]awsec2.ISecurityGroup{
		common.GetAllowAllVPCSecurityGroup(stack, jsii.String("SG"), jsii.String("Acorn generated RDS security group."), vpc, 3306),
	}

	creds := awsrds.Credentials_FromGeneratedSecret(jsii.String(props.AdminUser), &awsrds.CredentialsBaseOptions{})

	var parameterGroup awsrds.ParameterGroup
	if len(props.Parameters) > 0 {
		parameterGroup = rds.NewParameterGroup(stack, jsii.String("ParameterGroup"), props, engine)
	}

	cluster := awsrds.NewServerlessCluster(stack, jsii.String("Cluster"), &awsrds.ServerlessClusterProps{
		Engine:              engine,
		DefaultDatabaseName: jsii.String(props.DatabaseName),
		CopyTagsToSnapshot:  jsii.Bool(true),
		DeletionProtection:  jsii.Bool(props.DeletionProtection),
		RemovalPolicy:       rds.GetRemovalPolicy(props),

		Credentials: creds,
		Vpc:         vpc,
		Scaling: &awsrds.ServerlessScalingOptions{
			AutoPause:   awscdk.Duration_Minutes(jsii.Number(props.AutoPauseDurationMinutes)),
			MinCapacity: getACUFromInt(props.AuroraCapacityUnitsMin),
			MaxCapacity: getACUFromInt(props.AuroraCapacityUnitsMax),
		},
		SubnetGroup:    subnetGroup,
		SecurityGroups: sgs,
		ParameterGroup: parameterGroup,
	})

	if props.RestoreSnapshotArn != "" {
		awscdk.Aspects_Of(cluster).Add(rds.NewSnapshotAspect(props.RestoreSnapshotArn))
	}

	port := "3306"
	pSlice := strings.SplitN(*cluster.ClusterEndpoint().SocketAddress(), ":", 2)
	if len(pSlice) == 2 {
		port = pSlice[1]
	}

	awscdk.NewCfnOutput(stack, jsii.String("host"), &awscdk.CfnOutputProps{
		Value: cluster.ClusterEndpoint().Hostname(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("port"), &awscdk.CfnOutputProps{
		Value: &port,
	})
	awscdk.NewCfnOutput(stack, jsii.String("adminusername"), &awscdk.CfnOutputProps{
		Value: creds.Username(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("adminpasswordarn"), &awscdk.CfnOutputProps{
		Value: cluster.Secret().SecretArn(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("clusterid"), &awscdk.CfnOutputProps{
		Value: cluster.ClusterIdentifier(),
	})

	return stack
}

func getACUFromInt(i int) awsrds.AuroraCapacityUnit {
	switch i {
	case 1:
		return awsrds.AuroraCapacityUnit_ACU_1
	case 2:
		return awsrds.AuroraCapacityUnit_ACU_2
	case 4:
		return awsrds.AuroraCapacityUnit_ACU_4
	case 8:
		return awsrds.AuroraCapacityUnit_ACU_8
	case 16:
		return awsrds.AuroraCapacityUnit_ACU_16
	case 32:
		return awsrds.AuroraCapacityUnit_ACU_32
	case 64:
		return awsrds.AuroraCapacityUnit_ACU_64
	case 128:
		return awsrds.AuroraCapacityUnit_ACU_128
	case 256:
		return awsrds.AuroraCapacityUnit_ACU_256
	case 384:
		return awsrds.AuroraCapacityUnit_ACU_384
	default:
		logrus.Fatalf("invalid ACU request must be 1, 2, 4, 8, 16, 32, 64, 128, 256, 384. Passed in: %d", i)
	}
	return ""
}

func main() {
	defer jsii.Close()

	app := common.NewAcornTaggedApp(nil)
	stackProps := &rds.RDSStackProps{
		StackProps: *common.NewAWSCDKStackProps(),
	}
	stackProps.VpcID = common.GetVpcID()

	if err := common.NewConfig(stackProps); err != nil {
		logrus.Fatal(err)
	}

	NewRDSStack(app, stackProps)

	app.Synth(nil)
}
