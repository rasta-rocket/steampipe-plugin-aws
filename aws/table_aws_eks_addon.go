package aws

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"

	"github.com/aws/aws-sdk-go/service/eks"
)

//// TABLE DEFINITION

func tableAwsEksAddon(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_eks_addon",
		Description: "AWS EKS Addon",
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.AllColumns([]string{"addon_name", "cluster_name"}),
			ShouldIgnoreError: isNotFoundError([]string{"ResourceNotFoundException", "InvalidParameterException", "InvalidParameter"}),
			Hydrate:           getEksAddon,
		},
		List: &plugin.ListConfig{
			ParentHydrate: listEksClusters,
			Hydrate:       listEksAddons,
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "addon_name",
				Description: "The name of the add-on.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "arn",
				Description: "The Amazon Resource Name (ARN) of the add-on.",
				Type:        proto.ColumnType_STRING,
				Hydrate:     getEksAddon,
				Transform:   transform.FromField("AddonArn"),
			},
			{
				Name:        "cluster_name",
				Description: "The name of the cluster.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "addon_version",
				Description: "The version of the add-on.",
				Type:        proto.ColumnType_STRING,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "status",
				Description: "The status of the add-on.",
				Type:        proto.ColumnType_STRING,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "created_at",
				Description: "The date and time that the add-on was created.",
				Type:        proto.ColumnType_TIMESTAMP,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "modified_at",
				Description: "The date and time that the add-on was last modified.",
				Type:        proto.ColumnType_TIMESTAMP,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "service_account_role_arn",
				Description: "The Amazon Resource Name (ARN) of the IAM role that is bound to the Kubernetes service account used by the add-on.",
				Type:        proto.ColumnType_STRING,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "health_issues",
				Description: "An object that represents the add-on's health issues.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getEksAddon,
				Transform:   transform.FromField("Health.Issues"),
			},

			// Steampipe standard columns
			{
				Name:        "title",
				Description: resourceInterfaceDescription("title"),
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("AddonName"),
			},
			{
				Name:        "tags",
				Description: "The metadata that you apply to the cluster to assist with categorization and organization.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getEksAddon,
			},
			{
				Name:        "akas",
				Description: resourceInterfaceDescription("akas"),
				Type:        proto.ColumnType_JSON,
				Hydrate:     getEksAddon,
				Transform:   transform.FromField("AddonArn").Transform(transform.EnsureStringArray),
			},
		}),
	}
}

//// LIST FUNCTION

func listEksAddons(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	var region string
	matrixRegion := plugin.GetMatrixItem(ctx)[matrixKeyRegion]
	if matrixRegion != nil {
		region = matrixRegion.(string)
	}
	plugin.Logger(ctx).Trace("listEksAddons", "AWS_REGION", region)

	// Get cluster details
	clusterName := *h.Item.(*eks.Cluster).Name

	// Create service
	svc, err := EksService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	err = svc.ListAddonsPages(
		&eks.ListAddonsInput{ClusterName: &clusterName},
		func(page *eks.ListAddonsOutput, b bool) bool {
			for _, addon := range page.Addons {
				d.StreamListItem(ctx, &eks.Addon{
					AddonName:   addon,
					ClusterName: &clusterName,
				})
			}
			return true
		},
	)
	return nil, err
}

//// HYDRATE FUNCTIONS

func getEksAddon(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getEksAddon")

	var region string
	matrixRegion := plugin.GetMatrixItem(ctx)[matrixKeyRegion]
	if matrixRegion != nil {
		region = matrixRegion.(string)
	}

	var clusterName, addonName string
	if h.Item != nil {
		clusterName = *h.Item.(*eks.Addon).ClusterName
		addonName = *h.Item.(*eks.Addon).AddonName
	} else {
		clusterName = d.KeyColumnQuals["cluster_name"].GetStringValue()
		addonName = d.KeyColumnQuals["addon_name"].GetStringValue()
	}

	// create service
	svc, err := EksService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	params := &eks.DescribeAddonInput{
		AddonName:   &addonName,
		ClusterName: &clusterName,
	}

	op, err := svc.DescribeAddon(params)
	if err != nil {
		return nil, err
	}

	return op.Addon, nil
}