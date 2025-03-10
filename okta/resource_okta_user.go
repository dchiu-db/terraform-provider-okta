package okta

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

// All profile properties here so we can do a diff against the config to see if any have changed before making the
// request or before erring due to an update on a user that is DEPROVISIONED. Since we have core user props coupled
// with group/user membership a few change requests go out in the Update function.
var profileKeys = []string{
	"city",
	"cost_center",
	"country_code",
	"custom_profile_attributes",
	"department",
	"display_name",
	"division",
	"email",
	"employee_number",
	"first_name",
	"honorific_prefix",
	"honorific_suffix",
	"last_name",
	"locale",
	"login",
	"manager",
	"manager_id",
	"middle_name",
	"mobile_phone",
	"nick_name",
	"organization",
	"postal_address",
	"preferred_language",
	"primary_phone",
	"profile_url",
	"second_email",
	"state",
	"street_address",
	"timezone",
	"title",
	"user_type",
	"zip_code",
	"password",
	"recovery_question",
	"recovery_answer",
}

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				// Supporting id and email based imports
				user, _, err := getOktaClientFromMetadata(m).User.GetUser(ctx, d.Id())
				if err != nil {
					return nil, err
				}
				d.SetId(user.Id)
				err = setAdminRoles(ctx, d, m)
				if err != nil {
					return nil, fmt.Errorf("failed to set user's roles: %v", err)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"admin_roles": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "User Okta admin roles - ie. ['APP_ADMIN', 'USER_ADMIN']",
				Deprecated:  "The `admin_roles` field is now deprecated for the resource `okta_user`, please replace all uses of this with: `okta_user_admin_roles`",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: elemInSlice(validAdminRoles),
				},
			},
			"skip_roles": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Do not populate user roles information (prevents additional API call)",
			},
			"city": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User city",
			},
			"cost_center": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User cost center",
			},
			"country_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User country code",
			},
			"custom_profile_attributes": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: stringIsJSON,
				StateFunc:        normalizeDataJSON,
				Description:      "JSON formatted custom attributes for a user. It must be JSON due to various types Okta allows.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == ""
				},
			},
			"department": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User department",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User display name, suitable to show end users",
			},
			"division": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User division",
			},
			"email": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "User primary email address",
				ValidateDiagFunc: stringIsEmail,
			},
			"employee_number": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User employee number",
			},
			"first_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User first name",
			},
			"group_memberships": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The groups that you want this user to be a part of. This can also be done via the group using the `users` property.",
				Elem:        &schema.Schema{Type: schema.TypeString},
				Deprecated:  "The `group_memberships` field is now deprecated for the resource `okta_user`, please replace all uses of this with: `okta_user_group_memberships`",
			},
			"honorific_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User honorific prefix",
			},
			"honorific_suffix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User honorific suffix",
			},
			"last_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User last name",
			},
			"locale": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User default location",
			},
			"login": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "User Okta login",
			},
			"manager": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Manager of User",
			},
			"manager_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Manager ID of User",
			},
			"middle_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User middle name",
			},
			"mobile_phone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User mobile phone number",
			},
			"nick_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User nickname",
			},
			"organization": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User organization",
			},
			"postal_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User mailing address",
			},
			"preferred_language": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User preferred language",
			},
			"primary_phone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User primary phone number",
			},
			"profile_url": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "User online profile (web page)",
				ValidateDiagFunc: stringIsURL(validURLSchemes...),
			},
			"second_email": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "User secondary email address, used for account recovery",
				ValidateDiagFunc: stringIsEmail,
			},
			"state": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User state or region",
			},
			"status": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The status of the User in Okta - remove to set user back to active/provisioned",
				Default:          statusActive,
				ValidateDiagFunc: elemInSlice([]string{statusActive, userStatusStaged, userStatusDeprovisioned, userStatusSuspended}),
				// ignore diff changing to ACTIVE if state is set to PROVISIONED or PASSWORD_EXPIRED
				// since this is a similar status in Okta terms
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == userStatusProvisioned && new == statusActive || old == userStatusPasswordExpired && new == statusActive
				},
			},
			"raw_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The raw status of the User in Okta - (status is mapped)",
			},
			"street_address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User street address",
			},
			"timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User default timezone",
			},
			"title": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User title",
			},
			"user_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User employee type",
			},
			"zip_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User zipcode or postal code",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "User Password",
			},
			"expire_password_on_create": {
				Type:         schema.TypeBool,
				Optional:     true,
				Default:      false,
				Description:  "If set to `true`, the user will have to change the password at the next login. This property will be used when user is being created and works only when `password` field is set",
				RequiredWith: []string{"password"},
			},
			"password_inline_hook": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: elemInSlice([]string{"default"}),
				Description:      "When specified, the Password Inline Hook is triggered to handle verification of the end user's password the first time the user tries to sign in",
				ConflictsWith:    []string{"password", "password_hash"},
			},
			"old_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Old User Password. Should be only set in case the password was not changed using the provider",
			},
			"recovery_question": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User Password Recovery Question",
			},
			"recovery_answer": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ValidateDiagFunc: stringLenBetween(4, 1000),
				Description:      "User Password Recovery Answer",
			},
			"password_hash": {
				Type:        schema.TypeSet,
				MaxItems:    1,
				Description: "Specifies a hashed password to import into Okta.",
				Optional:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					oldHash, newHash := d.GetChange("password_hash")
					if oldHash != nil && newHash != nil && len(oldHash.(*schema.Set).List()) > 0 && len(newHash.(*schema.Set).List()) > 0 {
						oh := oldHash.(*schema.Set).List()[0].(map[string]interface{})
						nh := newHash.(*schema.Set).List()[0].(map[string]interface{})
						return reflect.DeepEqual(oh, nh)
					}
					return new == "" || old == new
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"algorithm": {
							Description:      "The algorithm used to generate the hash using the password",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: elemInSlice([]string{"BCRYPT", "SHA-512", "SHA-256", "SHA-1", "MD5"}),
						},
						"work_factor": {
							Description:      "Governs the strength of the hash and the time required to compute it. Only required for BCRYPT algorithm",
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: intBetween(1, 20),
						},
						"salt": {
							Description: "Only required for salted hashes",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"salt_order": {
							Description:      "Specifies whether salt was pre- or postfixed to the password before hashing",
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: elemInSlice([]string{"PREFIX", "POSTFIX"}),
						},
						"value": {
							Description: "For SHA-512, SHA-256, SHA-1, MD5, This is the actual base64-encoded hash of the password (and salt, if used). This is the " +
								"Base64 encoded value of the SHA-512/SHA-256/SHA-1/MD5 digest that was computed by either pre-fixing or post-fixing the salt to the " +
								"password, depending on the saltOrder. If a salt was not used in the source system, then this should just be the the Base64 encoded " +
								"value of the password's SHA-512/SHA-256/SHA-1/MD5 digest. For BCRYPT, This is the actual radix64-encoded hashed password.",
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	logger(m).Info("creating user", "login", d.Get("login").(string))
	profile := populateUserProfile(d)
	qp := query.NewQueryParams()

	// setting activate to false on user creation will leave the user with a status of STAGED
	if d.Get("status").(string) == userStatusStaged {
		qp = query.NewQueryParams(query.WithActivate(false))
	}

	uc := &okta.UserCredentials{
		Password: &okta.PasswordCredential{
			Value: d.Get("password").(string),
			Hash:  buildPasswordCredentialHash(d.Get("password_hash")),
		},
	}
	pih := d.Get("password_inline_hook").(string)
	if pih != "" {
		uc.Password = &okta.PasswordCredential{
			Hook: &okta.PasswordCredentialHook{
				Type: pih,
			},
		}
	}
	recoveryQuestion := d.Get("recovery_question").(string)
	recoveryAnswer := d.Get("recovery_answer").(string)
	if recoveryQuestion != "" {
		uc.RecoveryQuestion = &okta.RecoveryQuestionCredential{
			Question: recoveryQuestion,
			Answer:   recoveryAnswer,
		}
	}

	userBody := okta.CreateUserRequest{
		Profile:     profile,
		Credentials: uc,
	}
	client := getOktaClientFromMetadata(m)
	user, _, err := client.User.CreateUser(ctx, userBody, qp)
	if err != nil {
		return diag.Errorf("failed to create user: %v", err)
	}
	// set the user id into state before setting roles and status in case they fail
	d.SetId(user.Id)

	// role assigning can only happen after the user is created so order matters here
	// Only sync admin roles when it is set by a consumer as field is deprecated
	if _, exists := d.GetOk("admin_roles"); exists {
		roles := convertInterfaceToStringSetNullable(d.Get("admin_roles"))
		if roles != nil {
			err = assignAdminRolesToUser(ctx, user.Id, roles, false, client)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Only sync when there is opt in, consumers can chose which route they want to take
	if _, exists := d.GetOk("group_memberships"); exists {
		groups := convertInterfaceToStringSetNullable(d.Get("group_memberships"))
		err = assignGroupsToUser(ctx, user.Id, groups, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// status changing can only happen after user is created as well
	if d.Get("status").(string) == userStatusSuspended || d.Get("status").(string) == userStatusDeprovisioned {
		err := updateUserStatus(ctx, user.Id, d.Get("status").(string), client)
		if err != nil {
			return diag.Errorf("failed to update user status: %v", err)
		}
	}

	expire, ok := d.GetOk("expire_password_on_create")
	if ok && expire.(bool) {
		_, _, err = getOktaClientFromMetadata(m).User.ExpirePassword(ctx, user.Id)
		if err != nil {
			return diag.Errorf("failed to expire user's password: %v", err)
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	logger(m).Info("reading user", "id", d.Id())
	client := getOktaClientFromMetadata(m)
	user, resp, err := client.User.GetUser(ctx, d.Id())
	if err := suppressErrorOn404(resp, err); err != nil {
		return diag.Errorf("failed to get user: %v", err)
	}
	if user == nil {
		d.SetId("")
		return nil
	}
	_ = d.Set("raw_status", user.Status)
	rawMap := flattenUser(user)
	err = setNonPrimitives(d, rawMap)
	if err != nil {
		return diag.Errorf("failed to set user's properties: %v", err)
	}
	if val := d.Get("skip_roles"); val != nil {
		if skip, ok := val.(bool); ok && !skip {
			err = setAdminRoles(ctx, d, m)
			if err != nil {
				return diag.Errorf("failed to set user's admin roles: %v", err)
			}
		}
	}

	// Only sync when it is outlined, an empty list will remove all membership
	if _, exists := d.GetOk("group_memberships"); exists {
		err = setGroupUserMemberships(ctx, d, client)
		if err != nil {
			return diag.Errorf("failed to set user's groups: %v", err)
		}
	}
	return nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	logger(m).Info("updating user", "id", d.Id())
	status := d.Get("status").(string)
	statusChange := d.HasChange("status")

	if status == userStatusStaged && statusChange {
		return diag.Errorf("Okta will not allow a user to be updated to STAGED. Can set to STAGED on user creation only")
	}

	// There are a few requests here so just making sure the state gets updated per successful downstream change
	roleChange := d.HasChange("admin_roles")
	groupChange := d.HasChange("group_memberships")
	userChange := hasProfileChange(d)
	passwordChange := d.HasChange("password")
	passwordHashChange := d.HasChange("password_hash")
	passwordHookChange := d.HasChange("password_inline_hook")
	recoveryQuestionChange := d.HasChange("recovery_question")
	recoveryAnswerChange := d.HasChange("recovery_answer")

	client := getOktaClientFromMetadata(m)
	if passwordChange {
		user, _, err := client.User.GetUser(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to get user: %v", err)
		}
		if user.Status == "PROVISIONED" {
			return diag.Errorf("can not change password for provisioned user, the activation workflow should be " +
				"finished first. Please, check this diagram https://developer.okta.com/docs/reference/api/users/#user-status for more clarity.")
		}
	}

	// run the update status func first so a user that was previously deprovisioned
	// can be updated further if it's status changed in it's terraform configs
	if statusChange {
		err := updateUserStatus(ctx, d.Id(), status, client)
		if err != nil {
			return diag.Errorf("failed to update user status: %v", err)
		}
		_ = d.Set("status", status)
	}

	if status == userStatusDeprovisioned && userChange {
		return diag.Errorf("Only the status of a DEPROVISIONED user can be updated, we detected other change")
	}

	if userChange || passwordHashChange || passwordHookChange {
		profile := populateUserProfile(d)
		userBody := okta.User{
			Profile: profile,
		}
		if passwordHashChange {
			userBody.Credentials = &okta.UserCredentials{
				Password: &okta.PasswordCredential{
					Hash: buildPasswordCredentialHash(d.Get("password_hash")),
				},
			}
		}
		pih := d.Get("password_inline_hook").(string)
		if passwordHookChange && pih != "" {
			userBody.Credentials = &okta.UserCredentials{
				Password: &okta.PasswordCredential{
					Hook: &okta.PasswordCredentialHook{
						Type: pih,
					},
				},
			}
		}
		_, _, err := client.User.UpdateUser(ctx, d.Id(), userBody, nil)
		if err != nil {
			return diag.Errorf("failed to update user: %v", err)
		}
	}

	if roleChange {
		oldRoles, newRoles := d.GetChange("admin_roles")
		oldSet := oldRoles.(*schema.Set)
		newSet := newRoles.(*schema.Set)
		rolesToAdd := convertInterfaceArrToStringArr(newSet.Difference(oldSet).List())
		rolesToRemove := convertInterfaceArrToStringArr(oldSet.Difference(newSet).List())
		roles, _, err := listUserOnlyRoles(ctx, client, d.Id())
		if err != nil {
			return diag.Errorf("failed to list user's roles: %v", err)
		}
		for _, role := range roles {
			if contains(rolesToRemove, role.Type) {
				resp, err := client.User.RemoveRoleFromUser(ctx, d.Id(), role.Id)
				if err := suppressErrorOn404(resp, err); err != nil {
					return diag.Errorf("failed to remove user's role: %v", err)
				}
			}
		}
		err = assignAdminRolesToUser(ctx, d.Id(), rolesToAdd, false, client)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if groupChange {
		oldGM, newGM := d.GetChange("group_memberships")
		oldSet := oldGM.(*schema.Set)
		newSet := newGM.(*schema.Set)
		groupsToAdd := convertInterfaceArrToStringArr(newSet.Difference(oldSet).List())
		groupsToRemove := convertInterfaceArrToStringArr(oldSet.Difference(newSet).List())
		err := addUserToGroups(ctx, client, d.Id(), groupsToAdd)
		if err != nil {
			return diag.FromErr(err)
		}
		err = removeUserFromGroups(ctx, client, d.Id(), groupsToRemove)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if passwordChange {
		oldPassword, newPassword := d.GetChange("password")
		old, oldPasswordExist := d.GetOk("old_password")
		if oldPasswordExist {
			oldPassword = old
		}
		if oldPasswordExist {
			op := &okta.PasswordCredential{
				Value: oldPassword.(string),
			}
			np := &okta.PasswordCredential{
				Value: newPassword.(string),
			}
			npr := &okta.ChangePasswordRequest{
				OldPassword: op,
				NewPassword: np,
			}
			_, _, err := client.User.ChangePassword(ctx, d.Id(), *npr, nil)
			if err != nil {
				return diag.Errorf("failed to update user's password: %v", err)
			}
		}
		if !oldPasswordExist {
			password, _ := newPassword.(string)
			user := okta.User{
				Credentials: &okta.UserCredentials{
					Password: &okta.PasswordCredential{
						Value: password,
					},
				},
			}
			_, _, err := client.User.UpdateUser(ctx, d.Id(), user, nil)
			if err != nil {
				return diag.Errorf("failed to set user's password: %v", err)
			}
		}
	}

	if recoveryQuestionChange || recoveryAnswerChange {
		nuc := &okta.UserCredentials{
			Password: &okta.PasswordCredential{
				Value: d.Get("password").(string),
			},
			RecoveryQuestion: &okta.RecoveryQuestionCredential{
				Question: d.Get("recovery_question").(string),
				Answer:   d.Get("recovery_answer").(string),
			},
		}
		_, _, err := client.User.ChangeRecoveryQuestion(ctx, d.Id(), *nuc)
		if err != nil {
			return diag.Errorf("failed to change user's password recovery question: %v", err)
		}
	}
	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	logger(m).Info("deleting user", "id", d.Id())
	err := ensureUserDelete(ctx, d.Id(), d.Get("status").(string), getOktaClientFromMetadata(m))
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func buildPasswordCredentialHash(rawPasswordHash interface{}) *okta.PasswordCredentialHash {
	if rawPasswordHash == nil || len(rawPasswordHash.(*schema.Set).List()) == 0 {
		return nil
	}
	passwordHash := rawPasswordHash.(*schema.Set).List()
	hash := passwordHash[0].(map[string]interface{})
	wf, _ := hash["work_factor"].(int)
	h := &okta.PasswordCredentialHash{
		Algorithm:  hash["algorithm"].(string),
		Value:      hash["value"].(string),
		WorkFactor: int64(wf),
	}
	h.Salt, _ = hash["salt"].(string)
	h.SaltOrder, _ = hash["salt_order"].(string)
	return h
}

// Checks whether any profile keys have changed, this is necessary since the profile is not nested. Also, necessary
// to give a sensible user readable error when they attempt to update a DEPROVISIONED user. Previously
// this error always occurred when you set a user's status to DEPROVISIONED.
func hasProfileChange(d *schema.ResourceData) bool {
	for _, k := range profileKeys {
		if d.HasChange(k) {
			return true
		}
	}
	return false
}

func ensureUserDelete(ctx context.Context, id, status string, client *okta.Client) error {
	// only deprovisioned users can be deleted fully from okta
	// make two passes on the user if they aren't deprovisioned already to deprovision them first
	passes := 2
	if status == userStatusDeprovisioned {
		passes = 1
	}
	for i := 0; i < passes; i++ {
		_, err := client.User.DeactivateOrDeleteUser(ctx, id, nil)
		if err != nil {
			return fmt.Errorf("failed to deprovision or delete user from Okta: %v", err)
		}
	}
	return nil
}

func mapStatus(currentStatus string) string {
	// PASSWORD_EXPIRED and RECOVERY are effectively ACTIVE for our purposes
	if currentStatus == userStatusPasswordExpired || currentStatus == userStatusRecovery {
		return statusActive
	}
	return currentStatus
}
