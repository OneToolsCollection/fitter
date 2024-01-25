package utils_test

import (
	"github.com/PxyUp/fitter/pkg/builder"
	"github.com/PxyUp/fitter/pkg/config"
	"github.com/PxyUp/fitter/pkg/logger"
	"github.com/PxyUp/fitter/pkg/parser"
	"github.com/PxyUp/fitter/pkg/references"
	"github.com/PxyUp/fitter/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type TestFormatterSuite struct {
	suite.Suite
}

func TestRun(t *testing.T) {
	suite.Run(t, new(TestFormatterSuite))
}

func (s *TestFormatterSuite) TestFormatter() {
	index := uint32(8)
	assert.Equal(s.T(), "TokenRef=my_token and TokenObjectRef=my_token Object=value kek {\"value\": \"value kek\"} Env=test 8", utils.Format("TokenRef={{{RefName=TokenRef}}} and TokenObjectRef={{{RefName=TokenObjectRef token}}} Object={{{value}}} {PL} Env={{{FromEnv=TEST_VAL}}} {INDEX}", builder.Object(map[string]builder.Jsonable{
		"value": builder.String("value kek"),
	}), &index))
}

func (s *TestFormatterSuite) TearDownSuite() {
	_ = os.Unsetenv("TEST_VAL")
}
func (s *TestFormatterSuite) SetupSuite() {
	references.SetReference(map[string]*config.Reference{
		"TokenRef": {
			ModelField: &config.ModelField{
				ConnectorConfig: &config.ConnectorConfig{
					ResponseType: config.Json,
					StaticConfig: &config.StaticConnectorConfig{
						Value: builder.Object(map[string]builder.Jsonable{
							"token": builder.String("my_token"),
						}).ToJson(),
					},
				},
				Model: &config.Model{
					BaseField: &config.BaseField{
						Type: config.String,
						Path: "token",
					},
				},
			},
		},
		"TokenObjectRef": {
			ModelField: &config.ModelField{
				ConnectorConfig: &config.ConnectorConfig{
					ResponseType: config.Json,
					StaticConfig: &config.StaticConnectorConfig{
						Value: builder.Object(map[string]builder.Jsonable{
							"token": builder.String("my_token"),
						}).ToJson(),
					},
				},
				Model: &config.Model{
					ObjectConfig: &config.ObjectConfig{
						Fields: map[string]*config.Field{
							"token": {
								BaseField: &config.BaseField{
									Type: config.String,
									Path: "token",
								},
							},
						},
					},
				},
			},
		},
	}, func(_ string, model *config.ModelField) (builder.Jsonable, error) {
		return parser.NewEngine(model.ConnectorConfig, logger.Null).Get(model.Model, nil, nil)
	})
	os.Setenv("TEST_VAL", "test")
}