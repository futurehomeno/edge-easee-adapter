---
updatedAt: 2026-10-16T11:38:28.000Z
agentTools:
  projectIndex: https://developer.easee.com/llms.txt
---

# Controlling the charger

Available commands and responses

# Commands

The commands below are available for you to issue commands to a charger.

| COMMAND ID | COMMAND NAME                     | DESCRIPTION                                                                                                                                                | ARGUMENTS                                                                                      |
| :--------- | :------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------- |
| 25         | AuthorizeCharging                | Provide authorization result                                                                                                                               | IDToken \[String], Status \[Integer]                                                           |
| 26         | DeauthorizeCharging              | Deauthorizes charging                                                                                                                                      | IDToken \[String], Status \[Integer]                                                           |
| 1          | Reboot                           | Reboot charger                                                                                                                                             |                                                                                                |
| 48         | SetDynamicChargerCurrent         | Set max charge current for this charger \[Amperes], use 0 to pause charging (for smart charging / volatile setting)                                        | Current\_Amp \[Integer], TTL (minutes) \[Integer]                                              |
| 22         | SetDynamicCircuitCurrent         | Set charge current for each phase \[Amperes], use 0 for all phases to pause charging (for smart charging / volatile setting)                               | CurrentP1 \[Double], CurrentP2 \[Double], CurrentP3 \[Double], TTL (minutes) \[Integer]        |
| 29         | SetEnabled                       | Set true to enable charger                                                                                                                                 | IsEnabled \[Boolean]                                                                           |
| 30         | SetLockCablePermanently          | Set cable lock state                                                                                                                                       | Locked \[Boolean]                                                                              |
| 47         | SetMaxChargerCurrent             | Set max charge current for this charger \[Amperes]. Non-volatile setting, stored in flash. Should only be used for permanent change to current             | Current\_Amp \[Integer]                                                                        |
| 24         | SetMaxCircuitCurrent             | Non-volatile max current for circuit \[Amperes]                                                                                                            | CurrentP1 \[Integer], CurrentP2 \[Integer], CurrentP3 \[Integer]                               |
| 11         | SetSmartCharging                 | Set smart charging mode                                                                                                                                    | SmartCharging \[Boolean]                                                                       |
| 4          | UpgradeFirmware                  | Upgrade firmware to the latest version using delta OTA                                                                                                     |                                                                                                |
| 21         | SetDynamicCurrentOfflineFallback | Sets the maximum circuit current to be used by the load balancer in case the charger goes offline \[Amperes]                                               | MaxDynamicCurrentP1 \[Integer], MaxDynamicCurrentP2 \[Integer], MaxDynamicCurrentP3 \[Integer] |
| 63         | OverrideSchedule                 | Overrides the schedule for the current session. Output current will not be limited by the schedule. Schedule is automatically resumed when EV disconnects. |                                                                                                |

Example: Command to SetDynamicChargerCurrent:

```json
// body
{
  "Id": 48,
  "ExpiresAt": "2023-05-04T17:00:23.6305199Z",
  "Arguments": [
    { "Id": 1, "Value": "13" },
    { "Id": 2, "Value": "5" }
  ]
}
```

> 48:SetDynamicChargerCurrent \{with 13 amps, TTL 5 mins}

This will set 13 Amps as the current limit for 5 minutes. After the TTL (5min) expires, the charger will fall back to its prior limits.

Publish the body (above) to [RabbitMQ using the .NET client library](https://www.rabbitmq.com/dotnet-api-guide.html#publishing) (example):

```json C#
IBasicProperties props = channel.CreateBasicProperties();
props.ContentType = "text/plain";
props.Headers = new Dictionary<string, object>();
props.Headers.Add("OriginApplication",  "<OperatorName>.Easee");

var exchangeName = "<from-connection-details>";
var routingKey = "EH123456.C.48";

channel.BasicPublish(
		exchange: exchangeName,
		routingKey: routingKey,
		basicProperties: props,
		body: body);
```

> Using the .NET RabbitMQ client library example to publish a message. (Can be any supported language)

The exchange name is as given in the connection details.
The routing key should be in the format `<SerialNumber>.C.<CommandId>` for example: `EH123456.C.48`, where:
`SerialNumber` is `EH123456`,
`C` denotes a `Command`,
with `CommandId` set to `48` `(SetDynamicChargerCurrent)`.

Typically, publishers will utilise utility methods to ensure the correct routing key format is used for all messages published to the AMQP stream.

# Command responses

When a command is sent such as the example above `48:SetDynamicChargerCurrent`, two things occur:

1. A command response is returned (if command was successfully delivered)
2. An observation with the newly observed values are also emitted from the charger

A command response looks like:

```json
{
    "ArrivalTime": "2023-06-08T06:59:17.0130000+00:00", // from header
    "CommandResponse": {
        "Comment": null,
        "DeliveredAt": "2023-06-08T06:59:16.7660000Z",
        "Id": 48,
        "ResultCode": null,
        "SerialNumber": "EH000281",
        "Ticks": 638218043566469000,
        "Timestamp": "2023-06-08T06:59:16.6468930Z",
        "WasAccepted": true
    },
    "RoutingKey": "EH000281.CR.48"
}
```

Of note, a consumer can use the routing key, and values to the properties `Id` and `Ticks` to track which commands were successfully sent. This can be used to create a retry mechanism for example - triggering a retry if a command response is not received within a specified timeframe, such as 60 seconds.

An emitted observation for this command looks like:

```json
{
  "ArrivalTime": "2023-06-08T06:59:17.0130000+00:00", // from header
  "Observation": {
    "Description": "Max current this charger is allowed to offer to car (A). Volatile [Double]",
    "Id": 48,
    "Name": "DynamicChargerCurrent",
    "Timestamp": "2023-06-08T06:59:16.0000000+00:00",
    "Value": 13 // amps
  },
  "RoutingKey": "EH000281.O.48"
}
```

Example: Consume the command responses and observations from [RabbitMQ using the .NET client library](https://www.rabbitmq.com/dotnet-api-guide.html#publishing) (example):

```json C#
var factory = new ConnectionFactory
{
    UserName = "UserName",
    Password = "Password",                
    HostName = "HostName",
    Port = 5672,
    Ssl = new SslOption("HostName", enabled: true)
};

using var connection = factory.CreateConnection();
using (var channel = connection.CreateModel())
{    
    var consumer = new EventingBasicConsumer(channel);
    consumer.Received += (model, eventArguments) =>
    {
        // Get serial number from routing key
        // Routing key format "EH000281.O.48"
        var routingKeyParts = eventArguments.RoutingKey.Split(".");
        var serailNumber = routingKeyParts[0];       
        var messageType = routingKeyParts[1];        
        var messageBody = Encoding.UTF8.GetString(eventArguments.Body.Span);
    };
    channel.BasicConsume(queue: "Operator_{operatorId}.Q",
                         autoAck: true,
                         consumer: consumer);
}
```