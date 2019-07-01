package scaleway

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

func resourceScalewayComputeInstanceIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayComputeInstanceIPCreate,
		Read:   resourceScalewayComputeInstanceIPRead,
		Update: resourceScalewayComputeInstanceIPUpdate,
		Delete: resourceScalewayComputeInstanceIPDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ip address",
			},
			"reverse": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The reverse dns for this IP",
			},
			"server_id": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The server associated with this ip",
				DiffSuppressFunc: suppressLocality,
			},
			"zone":       zoneSchema(),
			"project_id": projectIDSchema(),
		},
	}
}

func resourceScalewayComputeInstanceIPCreate(d *schema.ResourceData, m interface{}) error {
	instanceApi, zone, err := getInstanceAPIWithZone(d, m)
	if err != nil {
		return err
	}

	res, err := instanceApi.CreateIP(&instance.CreateIPRequest{
		Zone:         zone,
		Organization: d.Get("project_id").(string),
	})
	if err != nil {
		return err
	}

	reverse := d.Get("reverse").(string)
	if reverse != "" {
		_, err = instanceApi.UpdateIP(&instance.UpdateIPRequest{
			Zone:    zone,
			IPID:    res.IP.ID,
			Reverse: &reverse,
		})
		if err != nil {
			return err
		}
	}

	d.SetId(newZonedId(zone, res.IP.ID))

	serverID := expandID(d.Get("server_id"))
	if serverID != "" {
		_, err = instanceApi.AttachIP(&instance.AttachIPRequest{
			Zone:     zone,
			IPID:     res.IP.ID,
			ServerID: serverID,
		})
		if err != nil {
			return err
		}

	}
	return resourceScalewayComputeInstanceIPRead(d, m)
}

func resourceScalewayComputeInstanceIPRead(d *schema.ResourceData, m interface{}) error {
	instanceApi, zone, ID, err := getInstanceAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return err
	}

	res, err := instanceApi.GetIP(&instance.GetIPRequest{
		IPID: ID,
		Zone: zone,
	})

	if err != nil {
		// We check for 403 because instance API returns 403 for a deleted IP
		if is404Error(err) || is403Error(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("address", res.IP.Address.String())
	d.Set("zone", string(zone))
	d.Set("project_id", res.IP.Organization)
	d.Set("reverse", res.IP.Reverse)

	return nil
}

func resourceScalewayComputeInstanceIPUpdate(d *schema.ResourceData, m interface{}) error {
	instanceApi, zone, ID, err := getInstanceAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("reverse") {
		l.Debugf("updating IP %q reverse to %q\n", d.Id(), d.Get("reverse"))

		reverse := d.Get("reverse").(string)
		_, err = instanceApi.UpdateIP(&instance.UpdateIPRequest{
			Zone:    zone,
			IPID:    ID,
			Reverse: &reverse,
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("server_id") {
		serverID := expandID(d.Get("server_id"))
		if serverID != "" {
			_, err = instanceApi.AttachIP(&instance.AttachIPRequest{
				Zone:     zone,
				IPID:     ID,
				ServerID: serverID,
			})
		} else {
			_, err = instanceApi.DetachIP(&instance.DetachIPRequest{
				Zone: zone,
				IPID: ID,
			})
		}
		if err != nil {
			return err
		}
	}

	return resourceScalewayComputeInstanceIPRead(d, m)
}

func resourceScalewayComputeInstanceIPDelete(d *schema.ResourceData, m interface{}) error {
	instanceApi, zone, ID, err := getInstanceAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return err
	}

	err = instanceApi.DeleteIP(&instance.DeleteIPRequest{
		IPID: ID,
		Zone: zone,
	})

	if err != nil && !is404Error(err) {
		return err
	}

	return nil
}
