package cloudability

import (
	"time"
	"log"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/skyscrapr/cloudability-sdk-go/cloudability"
)

func dataSourceAccount() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAccountRead,
		Schema: map[string]*schema.Schema{
			"vendor_account_id": {
				Type: schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "12 digit string corresponding to your AWS account ID",
			},
			"vendor_key": {
				Type: schema.TypeString,
				Optional: true,
				Default: "aws",
				ForceNew: true,
				Description: "'aws'",
			},
			"state": &schema.Schema {
				Type: schema.TypeString,
				Computed: true,
				Description: "Examples: unverified, verified, error",
			},
			"last_verification_attempted_at": &schema.Schema {
				Type: schema.TypeString,
				Computed: true,
				Description: "Date timestamp, example: 1970-01-01T00:00:00.000Z",
			},
			"message": &schema.Schema {
				Type: schema.TypeString,
				Computed: true,
				Description: "Error message for credentials in error state",
			},
			"retry_count": &schema.Schema {
				Type: schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default: 20,
				Description: "Number of times to retry the verification",
			},
			"retry_wait": &schema.Schema {
				Type: schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default: 5,
				Description: "Number of seconds to wait between verification retries",
			},
		},
	}
}

func dataSourceAccountRead(d *schema.ResourceData, meta interface{}) error {
	vendorKey := d.Get("vendor_key").(string)
	accountId := d.Get("vendor_account_id").(string)
	retryCount := d.Get("retry_count").(int)
	retryWait := d.Get("retry_wait").(int)
	
	client := meta.(*cloudability.CloudabilityClient)
	var account *cloudability.Account
	log.Printf("[DEBUG] resourceAccountVerificationCreate NewVerification [account_id: %q]", accountId)
    err := retry(retryCount, time.Duration(retryWait)*time.Second, func() (err error, exit bool) {
		account, err = client.Vendors.VerifyAccount(vendorKey, accountId)
		if err != nil {
			log.Printf("[DEBUG] VerifyAccount failed (%s)", err)
			return err, false
		} 
		if account.Verification.State == "error" {
			log.Printf("[DEBUG] Error verfifying account. Reason: %s", account.Verification.Message)
			err = fmt.Errorf("Verification was not successful: [%s] - %s", account.Verification.State, account.Verification.Message)
			return err, true
		} else if account.Verification.State != "verified" {
			log.Printf("[DEBUG] Invalid verfification state (%s) Reason: %s", account.Verification.State, account.Verification.Message)
			err = fmt.Errorf("Verification was not successful: [%s] - %s", account.Verification.State, account.Verification.Message)
		} else {
			log.Print("[DEBUG] Account Verified")
			return nil, true
		}
    	return err, false
    })
	if err != nil {
		log.Printf("[DEBUG] Could not verify the account: %q", err)
		return err
	}
	if account != nil {
		d.Set("vendor_account_id", account.VendorAccountId)
		d.Set("vendor_key", account.VendorKey)
		d.Set("state", account.Verification.State)
		d.Set("last_verification_attempted_at", account.Verification.LastVerificationAttemptedAt)
		d.Set("message", account.Verification.Message)
		d.SetId(account.Id)
	}
	return nil
}
