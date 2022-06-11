package ecs

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceStandaloneTask() *schema.Resource {
	return &schema.Resource{
		Create: resourceStandaloneTaskCreate,
		Read:   resourceStandaloneTaskRead,
		Update: resourceStandaloneTaskUpdate,
		Delete: resourceStandaloneTaskDelete,
		Importer: &schema.ResourceImporter{
			State: resourceStandaloneTaskImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"capacity_provider_strategy": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"base": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 100000),
						},
						"capacity_provider": {
							Type:     schema.TypeString,
							Required: true,
						},
						"weight": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 1000),
						},
					},
				},
			},
			"cluster": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"desired_count": {
				Type:     schema.TypeInt,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Get("scheduling_strategy").(string) == ecs.SchedulingStrategyDaemon
				},
			},
			"enable_ecs_managed_tags": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"enable_execute_command": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"force_new_deployment": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"group": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"launch_type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(ecs.LaunchType_Values(), false),
			},
			"network_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"assign_public_ip": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"security_groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
			"overrides": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_overrides": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"command": {
										Type:     schema.TypeList,
										Optional: true,
										ForceNew: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"cpu": {
										Type:     schema.TypeInt,
										Optional: true,
										ForceNew: true,
									},
									"environment": {
										Type:     schema.TypeMap,
										Optional: true,
										ForceNew: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"environmentFiles": {
										Type:     schema.TypeList,
										Optional: true,
										ForceNew: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringInSlice(ecs.EnvironmentFileType_Values(), false),
												},
												"value": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: verify.ValidARN,
												},
											},
										},
									},
									"memory": {
										Type:     schema.TypeInt,
										Optional: true,
										ForceNew: true,
									},
									"memory_reservation": {
										Type:     schema.TypeInt,
										Optional: true,
										ForceNew: true,
									},
									"name": {
										Type:     schema.TypeString,
										ForceNew: true,
										Required: true,
										Computed: true,
									},
									"resource_requirements": {
										Type:     schema.TypeList,
										Optional: true,
										ForceNew: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"type": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringInSlice(ecs.ResourceType_Values(), false),
												},
												"value": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},
						"cpu": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"ephemeral_storage": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"size_in_gib": {
										Type:         schema.TypeInt,
										Required:     true,
										ForceNew:     true,
										ValidateFunc: validation.IntBetween(21, 200),
									},
								},
							},
						},
						"execution_role_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      nil,
							ValidateFunc: verify.ValidARN,
						},
						"inference_accelerator_overrides": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"device_name": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"device_type": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
								},
							},
						},
						"memory": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"task_role_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: verify.ValidARN,
							ForceNew:     true,
						},
					},
				},
			},
			"task_definition": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"wait_for_steady_state": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},

		CustomizeDiff: customdiff.Sequence(
			verify.SetTagsDiff,
		),
	}
}

func resourceStandaloneTaskCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).ECSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	input := ecs.RunTaskInput{}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS()) // tags field doesn't exist in all partitions
	}

	// TODO: custominze input with all the parameters

	output, err := retryStandaloneTaskCreate(conn, input)
	if err != nil {
		return fmt.Errorf("failed creating ECS standalone task (%s): %w", d.Get("name").(string), err)
	}

	if output == nil || len(output.Tasks) == 0 {
		return fmt.Errorf("error creating ECS standalone task: empty response")
	}

	// TODO: update the state based on what was created

	return nil
}

func retryStandaloneTaskCreate(conn *ecs.ECS, input ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	var output *ecs.RunTaskOutput
	err := resource.Retry(propagationTimeout+serviceCreateTimeout, func() *resource.RetryError {
		var err error
		output, err = conn.RunTask(&input)

		if err != nil {
			if tfawserr.ErrCodeEquals(err, ecs.ErrCodeClusterNotFoundException) {
				return resource.RetryableError(err)
			}

			if tfawserr.ErrMessageContains(err, ecs.ErrCodeInvalidParameterException, "verify that the ECS service role being passed has the proper permissions") {
				return resource.RetryableError(err)
			}

			if tfawserr.ErrMessageContains(err, ecs.ErrCodeInvalidParameterException, "does not have an associated load balancer") {
				return resource.RetryableError(err)
			}

			if tfawserr.ErrMessageContains(err, ecs.ErrCodeInvalidParameterException, "Unable to assume the service linked role") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = conn.RunTask(&input)
	}

	return output, err
}

func resourceStandaloneTaskRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceStandaloneTaskUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceStandaloneTaskDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceStandaloneTaskImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return nil, nil
}
