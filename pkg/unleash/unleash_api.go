package unleash

import "time"

// Content of this file is copied
// from github.com/Unleash/unleash-client-go/v3/api

type Feature struct {
	// Name is the name of the feature toggle.
	Name string `json:"name"`

	// Description is a description of the feature toggle.
	Description string `json:"description"`

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool `json:"enabled"`

	// Strategies is a list of names of the strategies supported by the client.
	Strategies []Strategy `json:"strategies"`

	// CreatedAt is the creation time of the feature toggle.
	CreatedAt time.Time `json:"createdAt"`

	// Strategy is the strategy of the feature toggle.
	Strategy string `json:"strategy"`

	// Parameters is the parameters of the feature toggle.
	Parameters ParameterMap `json:"parameters"`

	// Variants is a list of variants of the feature toggle.
	Variants []VariantInternal `json:"variants"`
}

type Strategy struct {
	// Id is the name of the strategy.
	Id int `json:"id"`

	// Name is the name of the strategy.
	Name string `json:"name"`

	// Constraints is the constraints of the strategy.
	Constraints []Constraint `json:"constraints"`

	// Parameters is the parameters of the strategy.
	Parameters ParameterMap `json:"parameters"`
}

type Constraint struct {
	// ContextName is the context name of the constraint.
	ContextName string `json:"contextName"`

	// Operator is the operator of the constraint.
	Operator Operator `json:"operator"`

	// Values is the list of target values for multi-valued constraints.
	Values []string `json:"values"`

	// Value is the target value single-value constraints.
	Value string `json:"value"`

	// CaseInsensitive makes the string operators case-insensitive.
	CaseInsensitive bool `json:"caseInsensitive"`

	// Inverted flips the constraint check result.
	Inverted bool `json:"inverted"`
}

type Operator string

type VariantInternal struct {
	Variant
	// Weight is the traffic ratio for the request
	Weight int `json:"weight"`
	// WeightType can be fixed or variable
	WeightType string `json:"weightType"`
	// Override is used to get a variant accoording to the Unleash context field
	Overrides []Override `json:"overrides"`
}
type Override struct {
	// ContextName is the value of attribute context name
	ContextName string `json:"contextName"`
	// Values is the value of attribute values
	Values []string `json:"values"`
}

type Variant struct {
	// Name is the value of the variant name.
	Name string `json:"name"`
	// Payload is the value of the variant payload
	Payload Payload `json:"payload"`
	// Enabled indicates whether the feature which is extend by this variant was enabled or not.
	Enabled bool `json:"enabled"`
}

type Payload struct {
	// Type is the type of the payload
	Type string `json:"type"`
	// Value is the value of the payload type
	Value string `json:"value"`
}

type ParameterMap map[string]interface{}
