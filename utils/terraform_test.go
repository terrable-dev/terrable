package utils

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
)

func TestParseModuleConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		hclContent         string
		wantGlobalTimeout  int
		wantHandlerTimeout int
		wantErr            bool
	}{
		{
			name: "uses default timeout when not specified",
			hclContent: `
                module "test" {
                    handlers = {
                        TestHandler = {
                            source = "./test.ts"
                            http = {
                                GET = "/test"
                            }
                        }
                    }
                }
            `,
			wantGlobalTimeout:  DefaultTimeout,
			wantHandlerTimeout: DefaultTimeout,
			wantErr:            false,
		},
		{
			name: "uses global timeout when specified",
			hclContent: `
                module "test" {
                    timeout = 5
                    handlers = {
                        TestHandler = {
                            source = "./test.ts"
                            http = {
                                GET = "/test"
                            }
                        }
                    }
                }
            `,
			wantGlobalTimeout:  5,
			wantHandlerTimeout: 5,
			wantErr:            false,
		},
		{
			name: "handler timeout overrides global timeout",
			hclContent: `
                module "test" {
                    timeout = 5
                    handlers = {
                        TestHandler = {
                            timeout = 10
                            source = "./test.ts"
                            http = {
                                GET = "/test"
                            }
                        }
                    }
                }
            `,
			wantGlobalTimeout:  5,
			wantHandlerTimeout: 10,
			wantErr:            false,
		},
		{
			name: "invalid timeout value returns error",
			hclContent: `
                module "test" {
                    timeout = "invalid"
                    handlers = {
                        TestHandler = {
                            source = "./test.ts"
                            http = {
                                GET = "/test"
                            }
                        }
                    }
                }
            `,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tt.hclContent), "test.tf", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("failed to parse HCL: %s", diags.Error())
			}

			content, _ := file.Body.Content(&hcl.BodySchema{
				Blocks: []hcl.BlockHeaderSchema{
					{Type: "module", LabelNames: []string{"name"}},
				},
			})

			moduleBlock := content.Blocks[0]
			config, err := ParseModuleConfiguration("test.tf", moduleBlock)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, config)
			assert.Equal(t, tt.wantGlobalTimeout, config.Timeout, "Global timeout mismatch")

			if len(config.Handlers) > 0 {
				assert.Equal(t, tt.wantHandlerTimeout, config.Handlers[0].Timeout, "Handler timeout mismatch")
			}
		})
	}
}
