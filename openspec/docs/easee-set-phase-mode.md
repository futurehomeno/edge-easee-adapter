---
updatedAt: 2026-05-08T10:57:51.000Z
agentTools:
  projectIndex: https://developer.easee.com/llms.txt
---

# Set Phase Mode

Sets the Phase mode (1=locked to 1 phase, 2=auto, 3=locked to 3 phase, null sets it to 1 phase)

# OpenAPI definition

```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "Easee Cloud API",
    "description": "OpenAPI Definitions for Easee Cloud APIs.\n",
    "version": "v1"
  },
  "servers": [
    {
      "url": "https://api.easee.com"
    }
  ],
  "security": [
    {
      "Bearer": []
    }
  ],
  "tags": [
    {
      "name": "Charger",
      "description": "Read, configure and control chargers.",
      "x-displayName": "Charger"
    }
  ],
  "paths": {
    "/api/chargers/{chargerId}/commands/set_phase_mode": {
      "post": {
        "tags": [
          "Charger"
        ],
        "summary": "Set Phase Mode",
        "description": "Sets the Phase mode (1=locked to 1 phase, 2=auto, 3=locked to 3 phase, null sets it to 1 phase)",
        "operationId": "charger_set_phase_mode",
        "parameters": [
          {
            "name": "chargerId",
            "in": "path",
            "required": true,
            "description": "Charger serial number",
            "schema": {
              "type": "string"
            },
            "x-position": 1
          }
        ],
        "requestBody": {
          "x-name": "data",
          "description": "PhaseMode",
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/Easee_Command_API_ChargerPhaseModeDTO"
              }
            }
          },
          "required": true,
          "x-position": 2
        },
        "responses": {
          "200": {
            "description": "",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Easee_Command_API_Command"
                }
              }
            }
          },
          "400": {
            "description": "",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                }
              }
            }
          },
          "404": {
            "description": "",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Easee_Command_API_Command": {
        "type": "object",
        "additionalProperties": false,
        "required": [
          "id",
          "serialNumber",
          "timestamp"
        ],
        "properties": {
          "id": {
            "type": "integer",
            "format": "int32",
            "maximum": 255,
            "minimum": 1
          },
          "serialNumber": {
            "type": "string",
            "maxLength": 32,
            "minLength": 1
          },
          "timestamp": {
            "type": "string",
            "format": "date-time",
            "minLength": 1
          },
          "expiresAt": {
            "type": "string",
            "format": "date-time",
            "nullable": true
          },
          "originApplication": {
            "type": "string",
            "maxLength": 512,
            "minLength": 0,
            "nullable": true
          },
          "originReference": {
            "type": "string",
            "maxLength": 512,
            "minLength": 0,
            "nullable": true
          },
          "originAccount": {
            "type": "string",
            "maxLength": 512,
            "minLength": 0,
            "nullable": true
          },
          "deliveredAt": {
            "type": "string",
            "format": "date-time",
            "nullable": true
          },
          "wasAccepted": {
            "type": "boolean",
            "nullable": true
          },
          "arguments": {
            "type": "array",
            "nullable": true,
            "items": {
              "$ref": "#/components/schemas/Easee_Command_API_Argument"
            }
          },
          "resultCode": {
            "type": "integer",
            "format": "int32",
            "nullable": true
          },
          "comment": {
            "type": "string",
            "maxLength": 65535,
            "minLength": 0,
            "nullable": true
          },
          "status": {
            "$ref": "#/components/schemas/Easee_Command_API_CommandStatus"
          },
          "transport": {
            "nullable": true,
            "oneOf": [
              {
                "$ref": "#/components/schemas/Easee_Command_API_Transport"
              }
            ]
          }
        }
      },
      "Easee_Command_API_Argument": {
        "type": "object",
        "additionalProperties": false,
        "required": [
          "value"
        ],
        "properties": {
          "id": {
            "type": "integer",
            "format": "int32",
            "maximum": 65535,
            "minimum": 1
          },
          "value": {
            "type": "string",
            "maxLength": 65535,
            "minLength": 0
          }
        }
      },
      "Easee_Command_API_CommandStatus": {
        "type": "integer",
        "description": "",
        "x-enumNames": [
          "Sent",
          "Expired",
          "Executed",
          "Rejected"
        ],
        "enum": [
          0,
          1,
          2,
          3
        ]
      },
      "Easee_Command_API_Transport": {
        "type": "integer",
        "description": "",
        "x-enumNames": [
          "MQTT",
          "SMS"
        ],
        "enum": [
          1,
          2
        ]
      },
      "Easee_Command_API_ChargerPhaseModeDTO": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "phaseMode": {
            "type": "integer",
            "format": "int32",
            "nullable": true
          }
        }
      }
    },
    "securitySchemes": {
      "Bearer": {
        "type": "apiKey",
        "description": "JWT Authorization header using the Bearer scheme. Example: \"Authorization: Bearer {token}\"",
        "name": "Authorization",
        "in": "header"
      }
    }
  }
}
```