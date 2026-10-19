package llm

import (
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const categoryEnumJSON = `["еда", "транспорт", "жильё", "развлечения", "здоровье", "покупки", "другое"]`

type addExpenseArgs struct {
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     string  `json:"date"`
	Comment  string  `json:"comment"`
}

type listArgs struct {
	Period   string `json:"period"`
	From     string `json:"from"`
	To       string `json:"to"`
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

type updateLatestArgs struct {
	Amount     *float64 `json:"amount"`
	Category   string   `json:"category"`
	Date       string   `json:"date"`
	Comment    *string  `json:"comment"`
	commentSet bool
}

func (a *updateLatestArgs) UnmarshalJSON(data []byte) error {
	type alias updateLatestArgs
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = updateLatestArgs(tmp)
	_, a.commentSet = raw["comment"]
	return nil
}

type updateArgs struct {
	ID         string   `json:"id"`
	Amount     *float64 `json:"amount"`
	Category   string   `json:"category"`
	Date       string   `json:"date"`
	Comment    *string  `json:"comment"`
	commentSet bool
}

func (a *updateArgs) UnmarshalJSON(data []byte) error {
	type alias updateArgs
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = updateArgs(tmp)
	_, a.commentSet = raw["comment"]
	return nil
}

type deleteArgs struct {
	ID          string `json:"id"`
	CommentHint string `json:"comment_hint"`
	Category    string `json:"category"`
}

func tools() []openai.Tool {
	listParams := json.RawMessage(fmt.Sprintf(`{
		"type": "object",
		"properties": {
			"period": {
				"type": "string",
				"description": "today|yesterday|week|month|custom"
			},
			"from": {"type": "string", "description": "YYYY-MM-DD для custom"},
			"to": {"type": "string", "description": "YYYY-MM-DD для custom"},
			"category": {"type": "string", "enum": %s},
			"limit": {"type": "integer", "description": "макс. число записей для list"}
		}
	}`, categoryEnumJSON))

	updateFields := fmt.Sprintf(`{
		"amount": {"type": "number"},
		"category": {"type": "string", "enum": %s},
		"date": {"type": "string", "description": "YYYY-MM-DD, today или yesterday"},
		"comment": {"type": "string"}
	}`, categoryEnumJSON)

	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "add_expense",
				Description: "Сохранить трату пользователя",
				Parameters: json.RawMessage(fmt.Sprintf(`{
					"type": "object",
					"properties": {
						"amount": {"type": "number", "description": "Сумма в рублях (например 350 или 12.50)"},
						"category": {"type": "string", "enum": %s},
						"date": {"type": "string", "description": "YYYY-MM-DD, today или yesterday"},
						"comment": {"type": "string", "description": "Краткое описание, например кофе"}
					},
					"required": ["amount", "category"]
				}`, categoryEnumJSON)),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_expenses",
				Description: "Показать список трат пользователя с фильтрами",
				Parameters:  listParams,
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "sum_expenses",
				Description: "Посчитать сумму трат за период и/или категорию",
				Parameters:  listParams,
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_latest_expense",
				Description: "Изменить последнюю трату пользователя",
				Parameters: json.RawMessage(fmt.Sprintf(`{
					"type": "object",
					"properties": %s
				}`, updateFields)),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_expense",
				Description: "Изменить трату по id",
				Parameters: json.RawMessage(fmt.Sprintf(`{
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"amount": {"type": "number"},
						"category": {"type": "string", "enum": %s},
						"date": {"type": "string"},
						"comment": {"type": "string"}
					},
					"required": ["id"]
				}`, categoryEnumJSON)),
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "delete_expense",
				Description: "Удалить трату по id или последнюю подходящую по comment_hint/category",
				Parameters: json.RawMessage(fmt.Sprintf(`{
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"comment_hint": {"type": "string", "description": "например такси или кофе"},
						"category": {"type": "string", "enum": %s}
					}
				}`, categoryEnumJSON)),
			},
		},
	}
}
