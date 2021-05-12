package codefresh

import (
	"log"

	cfClient "github.com/codefresh-io/terraform-provider-codefresh/client"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	contextConfig     = "config"
	contextSecret     = "secret"
	contextYaml       = "yaml"
	contextSecretYaml = "secret-yaml"
)

var supportedContextType = []string{
	contextConfig,
	contextSecret,
	contextYaml,
	contextSecretYaml,
}

func getConflictingContexts(context string) []string {
	var conflictingTypes []string
	normalizedContext := normalizeFieldName(context)
	for _, value := range supportedContextType {
		normlizedValue := normalizeFieldName(value)
		if normlizedValue != normalizedContext {
			conflictingTypes = append(conflictingTypes, "spec.0."+normlizedValue)
		}
	}
	return conflictingTypes
}

func resourceContext() *schema.Resource {
	return &schema.Resource{
		Create: resourceContextCreate,
		Read:   resourceContextRead,
		Update: resourceContextUpdate,
		Delete: resourceContextDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"spec": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						normalizeFieldName(contextConfig): {
							Type:          schema.TypeList,
							ForceNew:      true,
							Optional:      true,
							MaxItems:      1,
							ConflictsWith: getConflictingContexts(contextConfig),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:     schema.TypeMap,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						normalizeFieldName(contextSecret): {
							Type:          schema.TypeList,
							Optional:      true,
							ForceNew:      true,
							MaxItems:      1,
							ConflictsWith: getConflictingContexts(contextSecret),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:      schema.TypeMap,
										Required:  true,
										Sensitive: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
						normalizeFieldName(contextYaml): {
							Type:          schema.TypeList,
							Optional:      true,
							ForceNew:      true,
							MaxItems:      1,
							ConflictsWith: getConflictingContexts(contextYaml),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:             schema.TypeString,
										Required:         true,
										ValidateFunc:     stringIsYaml,
										DiffSuppressFunc: suppressEquivalentYamlDiffs,
										StateFunc: func(v interface{}) string {
											template, _ := normalizeYamlString(v)
											return template
										},
									},
								},
							},
						},
						normalizeFieldName(contextSecretYaml): {
							Type:          schema.TypeList,
							Optional:      true,
							ForceNew:      true,
							MaxItems:      1,
							ConflictsWith: getConflictingContexts(contextSecretYaml),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:             schema.TypeString,
										Required:         true,
										Sensitive:        true,
										ValidateFunc:     stringIsYaml,
										DiffSuppressFunc: suppressEquivalentYamlDiffs,
										StateFunc: func(v interface{}) string {
											template, _ := normalizeYamlString(v)
											return template
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceContextCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*cfClient.Client)
	context := *mapResourceToContext(d)
	resp, err := client.CreateContext(&context)
	if err != nil {
		log.Printf("[DEBUG] Error while creating context. Error = %v", err)
		return err
	}

	d.SetId(resp.Metadata.Name)
	return resourceContextRead(d, meta)
}

func resourceContextRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cfClient.Client)

	contextName := d.Id()

	if contextName == "" {
		d.SetId("")
		return nil
	}

	context, err := client.GetContext(contextName)
	if err != nil {
		log.Printf("[DEBUG] Error while getting context. Error = %v", contextName)
		return err
	}

	err = mapContextToResource(*context, d)
	if err != nil {
		log.Printf("[DEBUG] Error while mapping context to resource. Error = %v", err)
		return err
	}

	return nil
}

func resourceContextUpdate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*cfClient.Client)

	context := *mapResourceToContext(d)
	context.Metadata.Name = d.Id()

	_, err := client.UpdateContext(&context)
	if err != nil {
		log.Printf("[DEBUG] Error while updating context. Error = %v", err)
		return err
	}

	return resourceContextRead(d, meta)
}

func resourceContextDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*cfClient.Client)

	err := client.DeleteContext(d.Id())
	if err != nil {
		return err
	}

	return nil
}

func mapContextToResource(context cfClient.Context, d *schema.ResourceData) error {

	err := d.Set("name", context.Metadata.Name)
	if err != nil {
		return err
	}

	err = d.Set("spec", flattenContextSpec(context.Spec))
	if err != nil {
		log.Printf("[DEBUG] Failed to flatten Context spec = %v", context.Spec)
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

func flattenContextSpec(spec cfClient.ContextSpec) []interface{} {

	var res = make([]interface{}, 0)
	m := make(map[string]interface{})

	switch currentContextType := spec.Type; currentContextType {
	case contextConfig, contextSecret:
		m[normalizeFieldName(currentContextType)] = flattenContextConfig(spec)
	case contextYaml, contextSecretYaml:
		m[normalizeFieldName(currentContextType)] = flattenContextYaml(spec)
	default:
		log.Printf("[DEBUG] Invalid context type = %v", currentContextType)
		return nil
	}

	res = append(res, m)
	return res
}

func flattenContextConfig(spec cfClient.ContextSpec) []interface{} {
	var res = make([]interface{}, 0)
	m := make(map[string]interface{})
	m["data"] = spec.Data
	res = append(res, m)
	return res
}

func flattenContextYaml(spec cfClient.ContextSpec) []interface{} {
	var res = make([]interface{}, 0)
	m := make(map[string]interface{})
	data, err := yaml.Marshal(spec.Data)
	if err != nil {
		return nil
	}
	m["data"] = string(data)
	res = append(res, m)
	return res
}

func mapResourceToContext(d *schema.ResourceData) *cfClient.Context {

	var normalizedContextType string
	var normalizedContextData map[string]interface{}

	if data, ok := d.GetOk("spec.0." + normalizeFieldName(contextConfig) + ".0.data"); ok {
		normalizedContextType = contextConfig
		normalizedContextData = data.(map[string]interface{})
	} else if data, ok := d.GetOk("spec.0." + normalizeFieldName(contextSecret) + ".0.data"); ok {
		normalizedContextType = contextSecret
		normalizedContextData = data.(map[string]interface{})
	} else if data, ok := d.GetOk("spec.0." + normalizeFieldName(contextYaml) + ".0.data"); ok {
		normalizedContextType = contextYaml
		yaml.Unmarshal([]byte(data.(string)), &normalizedContextData)
	} else if data, ok := d.GetOk("spec.0." + normalizeFieldName(contextSecretYaml) + ".0.data"); ok {
		normalizedContextType = contextSecretYaml
		yaml.Unmarshal([]byte(data.(string)), &normalizedContextData)
	}

	context := &cfClient.Context{
		Metadata: cfClient.ContextMetadata{
			Name: d.Get("name").(string),
		},
		Spec: cfClient.ContextSpec{
			Type: normalizedContextType,
			Data: normalizedContextData,
		},
	}

	return context
}
