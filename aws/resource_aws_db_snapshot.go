package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsDbSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSnapshotCreate,
		Read:   resourceAwsDbSnapshotRead,
		Update: resourceAwsDbSnapshotUpdate,
		Delete: resourceAwsDbSnapshotDelete,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"db_snapshot_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"db_instance_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"allocated_storage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"db_snapshot_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"engine": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"iops": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"license_model": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"option_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"source_db_snapshot_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshot_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))
	dBInstanceIdentifier := d.Get("db_instance_identifier").(string)

	params := &rds.CreateDBSnapshotInput{
		DBInstanceIdentifier: aws.String(dBInstanceIdentifier),
		DBSnapshotIdentifier: aws.String(d.Get("db_snapshot_identifier").(string)),
		Tags:                 tags,
	}

	_, err := conn.CreateDBSnapshot(params)
	if err != nil {
		return fmt.Errorf("Error creating AWS DB Snapshot %s: %s", dBInstanceIdentifier, err)
	}
	d.SetId(d.Get("db_snapshot_identifier").(string))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"available"},
		Refresh:    resourceAwsDbSnapshotStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutRead),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDbSnapshotRead(d, meta)
}

func resourceAwsDbSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(d.Id()),
	}
	resp, err := conn.DescribeDBSnapshots(params)

	if isAWSErr(err, rds.ErrCodeDBSnapshotNotFoundFault, "") {
		log.Printf("[WARN] AWS DB Snapshot (%s) is already gone", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error describing AWS DB Snapshot %s: %s", d.Id(), err)
	}

	snapshot := resp.DBSnapshots[0]

	d.Set("allocated_storage", snapshot.AllocatedStorage)
	d.Set("availability_zone", snapshot.AvailabilityZone)
	d.Set("db_snapshot_arn", snapshot.DBSnapshotArn)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("engine", snapshot.Engine)
	d.Set("engine_version", snapshot.EngineVersion)
	d.Set("iops", snapshot.Iops)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("license_model", snapshot.LicenseModel)
	d.Set("option_group_name", snapshot.OptionGroupName)
	d.Set("port", snapshot.Port)
	d.Set("source_db_snapshot_identifier", snapshot.SourceDBSnapshotIdentifier)
	d.Set("source_region", snapshot.SourceRegion)
	d.Set("snapshot_type", snapshot.SnapshotType)
	d.Set("status", snapshot.Status)
	d.Set("vpc_id", snapshot.VpcId)
	if err := saveTagsRDS(conn, d, aws.StringValue(snapshot.DBSnapshotArn)); err != nil {
		log.Printf("[WARN] Failed to save tags for RDS Snapshot (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsDbSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	params := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(d.Id()),
	}
	_, err := conn.DeleteDBSnapshot(params)
	if isAWSErr(err, rds.ErrCodeDBSnapshotNotFoundFault, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting AWS DB Snapshot %s: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsDbSnapshotUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	arn := d.Get("db_snapshot_arn").(string)
	if d.HasChange("tags") {
		oldTagsRaw, newTagsRaw := d.GetChange("tags")
		oldTagsMap := oldTagsRaw.(map[string]interface{})
		newTagsMap := newTagsRaw.(map[string]interface{})
		createTags, removeTags := diffTagsRDS(tagsFromMapRDS(oldTagsMap), tagsFromMapRDS(newTagsMap))

		if len(removeTags) > 0 {
			removeTagKeys := make([]*string, len(removeTags))
			for i, removeTag := range removeTags {
				removeTagKeys[i] = removeTag.Key
			}

			input := &rds.RemoveTagsFromResourceInput{
				ResourceName: aws.String(arn),
				TagKeys:      removeTagKeys,
			}

			log.Printf("[DEBUG] Untagging DB Snapshot: %s", input)
			if _, err := conn.RemoveTagsFromResource(input); err != nil {
				return fmt.Errorf("error untagging DB Snapshot (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &rds.AddTagsToResourceInput{
				ResourceName: aws.String(arn),
				Tags:         createTags,
			}

			log.Printf("[DEBUG] Tagging DB Snapshot: %s", input)
			if _, err := conn.AddTagsToResource(input); err != nil {
				return fmt.Errorf("error tagging DB Snapshot (%s): %s", d.Id(), err)
			}
		}
	}

	return nil
}

func resourceAwsDbSnapshotStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).rdsconn

		opts := &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(d.Id()),
		}

		log.Printf("[DEBUG] DB Snapshot describe configuration: %#v", opts)

		resp, err := conn.DescribeDBSnapshots(opts)
		if isAWSErr(err, rds.ErrCodeDBSnapshotNotFoundFault, "") {
			return nil, "", nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("Error retrieving DB Snapshots: %s", err)
		}

		if len(resp.DBSnapshots) != 1 {
			return nil, "", fmt.Errorf("No snapshots returned for %s", d.Id())
		}

		snapshot := resp.DBSnapshots[0]

		return resp, *snapshot.Status, nil
	}
}
