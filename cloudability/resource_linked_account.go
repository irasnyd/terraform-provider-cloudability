package cloudability

import (
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/skyscrapr/cloudability-sdk-go/cloudability"
	"log"
)

func resourceLinkedAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceLinkedAccountCreate,
		Read:   resourceLinkedAccountRead,
		Delete: resourceLinkedAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"vendor_account_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name given to your AWS account",
			},
			"vendor_account_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "12 digit string corresponding to your AWS account ID",
			},
			"vendor_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "aws",
				ForceNew:    true,
				Description: "'aws'",
			},
			"verification": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Examples: unverified, verified, error",
						},
						"last_verification_attempted_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Date timestamp, example: 1970-01-01T00:00:00.000Z",
						},
						"message": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Error message for credentials in error state",
						},
					},
				},
				Description: "Object containing details of verification state",
			},
			"authorization": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "'aws_role' or 'aws_user'",
						},
						"role_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "currently hardcoded to 'CloudabilityRole'",
						},
						"external_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The external ID used to prevent confused deputies. Generated by Cloudability",
						},
					},
				},
				Description: "Object contain vendor specific authorization details",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "aws_role",
				ForceNew:    true,
				Description: "'aws_role' or 'aws_user'",
			},
			"external_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The external ID used to prevent confused deputies. Generated by Cloudability",
			},
			"parent_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "12 digit string representing parent's account ID (if current cred is a linked account)",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date timestamp corresponding to cloudability credential creation time",
			},
		},
	}
}

func resourceLinkedAccountCreate(d *schema.ResourceData, meta interface{}) error {
	vendorKey := d.Get("vendor_key").(string)
	accountID := d.Get("vendor_account_id").(string)
	credType := d.Get("type").(string)

	client := meta.(*cloudability.Client)
	log.Printf("[DEBUG] resourceAccountCreate NewAccount [account_id: %q]", accountID)
	params := &cloudability.NewLinkedAccountParams{
		VendorAccountID: accountID,
		Type:            credType,
	}
	_, err := client.Vendors().NewLinkedAccount(vendorKey, params)
	if err != nil {
		return err
	}
	return resourceLinkedAccountRead(d, meta)
}

func resourceLinkedAccountRead(d *schema.ResourceData, meta interface{}) error {
	vendorKey := d.Get("vendor_key").(string)
	accountID := d.Get("vendor_account_id").(string)
	client := meta.(*cloudability.Client)
	log.Printf("[DEBUG] resourceLinkedAccountRead [account_id: %q]", accountID)
	account, err := client.Vendors().GetAccount(vendorKey, accountID)
	if err != nil {
		// Ignore 404 errors (No account found)
		var apiError cloudability.APIError
		jsonErr := json.Unmarshal([]byte(err.Error()), &apiError)
		if jsonErr == nil && apiError.Error.Code == 404 {
			log.Print("[DEBUG] resourceLinkedAccountRead Account not found. Ignoring")
			err = nil
		} else {
			return err
		}
	}

	if account != nil {
		d.Set("vendor_account_name", account.VendorAccountName)
		d.Set("vendor_account_id", account.VendorAccountID)
		d.Set("vendor_key", account.VendorKey)
		d.Set("verification", flattenVerification(account.Verification))
		d.Set("authorization", flattenAuthorization(account.Authorization))
		d.Set("external_id", account.Authorization.ExternalID)
		d.Set("parent_account_id", account.ParentAccountID)
		d.Set("created_at", account.CreatedAt)
		d.SetId(account.ID)
	}
	return nil
}

func resourceLinkedAccountDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudability.Client)
	vendorKey := d.Get("vendor_key").(string)
	accountID := d.Get("vendor_account_id").(string)
	err := client.Vendors().DeleteAccount(vendorKey, accountID)
	return err
}
