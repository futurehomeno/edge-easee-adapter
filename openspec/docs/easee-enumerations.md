---
updatedAt: 2026-10-16T11:41:29.000Z
agentTools:
  projectIndex: https://developer.easee.com/llms.txt
---

# Error Codes / Enumerations

# Description

You can use our [Observations](https://developer.easee.com/reference/charger_getobservations_mock) endpoint to retrieve an observation for a specific charger within a specified time frame. (ISO8601).

Here’s a list of supported Observation Enumerations and what each of them means to help you troubleshoot your device's behaviour.

## Charger Colour

| ChargerColorType | Name       |
| :--------------- | :--------- |
| 1                | Black      |
| 2                | Red        |
| 3                | Blue       |
| 4                | White      |
| 5                | Anthracite |

## Offline Charging Mode (45)

| OfflineChargingModeType | Name          | Description                                                       |
| :---------------------- | :------------ | :---------------------------------------------------------------- |
| 0                       | Always        | Always allow charging if offline.                                 |
| 1                       | IfWhitelisted | Only allow charging if token is allowed in the local token cache. |
| 2                       | Never         | Never allow charging if offline.                                  |

## Op Mode (109)

| OpMode | Name                    | Description                                                                |
| :----- | :---------------------- | :------------------------------------------------------------------------- |
| 0      | Offline                 | Offline.                                                                   |
| 1      | Disconnected            | No car connected.                                                          |
| 2      | Awaiting Start          | Car connected, charger is waiting for EV or load balancing. SuspendedEVSE. |
| 3      | Charging                | Charging.                                                                  |
| 4      | Completed               | Car has paused/stopped charging.                                           |
| 5      | Error                   | Error in charger.                                                          |
| 6      | Ready to Charge         | Charger is waiting for car to take energy. SuspendedEV.                    |
| 7      | Awaiting Authentication | Charger is waiting for authentication.                                     |
| 8      | De-authenticating       | Charger is de-authenticating.                                              |

## Output Phase (110)

| OutputPhaseType | Name                   | Description                   |
| :-------------- | :--------------------- | :---------------------------- |
| 0               | UNASSIGNED             | Unassigned                    |
| 10              | P1\_T2\_T3\_TN         | 1-phase (N+L1)                |
| 11              | P1\_T2\_T3\_IT         | 1-phase (L1+L2)               |
| 12              | P1\_T2\_T4\_TN         | 1-phase (N+L2)                |
| 13              | P1\_T2\_T4\_IT         | 1-phase (L1+L3)               |
| 14              | P1\_T2\_T5\_TN         | 1-phase (N+L3)                |
| 15              | P1\_T3\_T4\_IT         | 1-phase (L2+L3)               |
| 20              | P2\_T2\_T3\_T4\_TN     | 2-phases on TN (N+L1, N+L2)   |
| 21              | P2\_T2\_T4\_T5\_TN     | 2-phases on TN (N+L2, N+L3)   |
| 22              | P2\_T2\_T3\_T4\_IT     | 2-phases on IT (L1+L2, L2+L3) |
| 30              | P3\_T2\_T3\_T4\_T5\_TN | 3-phases (N+L1, N+L2, N+L3)   |

## Pairing Result Code

| PairingResultCode | Name                     |
| :---------------- | :----------------------- |
| 0                 | Success                  |
| 1                 | WrongAccountType         |
| 2                 | TooManyAttempts          |
| 3                 | AlreadyPairedWithPartner |
| 4                 | IncorrectPIN             |
| 5                 | AlreadyPairedWithUser    |
| 6                 | NotPairedWithPartner     |
| 7                 | NotPairedWithUser        |

## ReasonForNoCurrent (96)

| Id  | Name                   | Description                                                                                                                                                                                                    |
| :-- | :--------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0   | Charger Fine           | Charger is OK, use main charger status                                                                                                                                                                         |
| 1   | Load balancing         | Max circuit current too low, adjust power circuit up.                                                                                                                                                          |
| 2   | Load balancing         | Max dynamic circuit current too low (Partner Load balancing)                                                                                                                                                   |
| 3   | Load balancing         | Max dynamic offline fallback circuit current too low.                                                                                                                                                          |
| 4   | Load balancing         | Circuit fuse too low.                                                                                                                                                                                          |
| 5   | Load balancing         | Waiting in queue.                                                                                                                                                                                              |
| 6   | Load balancing         | Waiting in fully charged queue (Assumes a connected EV uses delayed charging, EV Charging complete.                                                                                                            |
| 7   | Error                  | Illegal grid type (Error - Fault in automatic grid type detection)                                                                                                                                             |
| 8   | Error                  | Primary unit has not received current request from secondary unit (car)                                                                                                                                        |
| 9   | Error                  | Master communication lost (Error)                                                                                                                                                                              |
| 10  | Error                  | No current from Equalizer to low.                                                                                                                                                                              |
| 11  | Error                  | No current, phase not connected.                                                                                                                                                                               |
| 25  | Error                  | Current limited by circuit fuse.                                                                                                                                                                               |
| 26  | Error                  | Current limited by circuit max current.                                                                                                                                                                        |
| 27  | Error                  | Current limited by dynamic circuit current.                                                                                                                                                                    |
| 28  | Error                  | Current limited by Equalizer.                                                                                                                                                                                  |
| 29  | Error                  | Current limited by circuit load balancing.                                                                                                                                                                     |
| 30  | Error                  | Current limited by offline settings.                                                                                                                                                                           |
| 50  | Load balancing circuit | Secondary unit not requesting current (No car connected)                                                                                                                                                       |
| 51  | Load balancing circuit | Max charger current too low.                                                                                                                                                                                   |
| 52  | Load balancing circuit | Max Dynamic charger current too low                                                                                                                                                                            |
| 53  | Informational          | Charger disabled.                                                                                                                                                                                              |
| 54  | Waiting                | Pending scheduled charging                                                                                                                                                                                     |
| 55  | Waiting                | Pending authorization                                                                                                                                                                                          |
| 56  | Error                  | Charger in error state.                                                                                                                                                                                        |
| 57  | Error                  | Erratic EV                                                                                                                                                                                                     |
| 75  | Cable                  | Current limited by cable rating.                                                                                                                                                                               |
| 76  | Schedule               | Current limited by schedule.                                                                                                                                                                                   |
| 77  | Charger Limit          | Current limited by charger max current.                                                                                                                                                                        |
| 78  | Charger Limit          | Current limited by dynamic charger current.                                                                                                                                                                    |
| 79  | Car Limit              | Current limited by car not charging.                                                                                                                                                                           |
| 80  | Local Adjustment       | Current limited by local adjustment, this can happen if a limit is set (lowered and then raised again). If this remains for a short while it's due to the current ramping up, If it stays it's due to the car. |
| 81  | Car Limit              | Current limited by car.                                                                                                                                                                                        |
| 100 | UndefinedError         | Undefined Error                                                                                                                                                                                                |

## LED Mode (46)

| LED   | Name                       | Description                                  |
| :---- | :------------------------- | :------------------------------------------- |
| 0     | Charger Disabled           | Charger is disabled.                         |
| 1-8   | Charger Updating           | Charger is updating.                         |
| 9-15  | Charger Updating           | Charger is updating.                         |
| 16-17 | Charger Faulty             | Charger is faulty.                           |
| 18    | Standby Master             | Standby Master.                              |
| 19    | Standby Secondary          | Standby Secondary.                           |
| 20    | Secondary Unit Searching   | Secondary unit searching for master.         |
| 21    | Smart Mode (Not Charging)  | Smart mode (Not charging).                   |
| 22    | Smart Mode (Charging)      | Smart mode (Charging).                       |
| 23    | Normal Mode (Not Charging) | Normal mode (Not charging).                  |
| 24    | Normal Mode (Charging)     | Normal mode (Charging).                      |
| 25    | Waiting for Authorization  | Waiting for authorization.                   |
| 26    | Verifying with Backend     | Verifying with backend.                      |
| 27    | Check Configuration        | Check configuration (Backplate chip defect). |
| 29    | Pairing RFID Keys          | Pairing RFID Keys.                           |
| 43-44 | Self Test Mode             | Self test mode.                              |

## PhaseMode (38)

| PhaseMode | Name                           |
| :-------- | :----------------------------- |
| 0         | Ignore, no phase mode reported |
| 1         | Locked to 1-phase              |
| 2         | Auto phase mode                |
| 3         | Locked to 3-phase              |

## PilotMode (100)

| PilotMode | Name                  | Description                                       |
| :-------- | :-------------------- | :------------------------------------------------ |
| A         | Car Disconnected      | Car disconnected.                                 |
| B         | Car Connected         | Car connected.                                    |
| C         | Car Charging          | Car charging.                                     |
| D         | Car Needs Ventilation | Car needs ventilation.                            |
| F         | Fault Detected        | Fault detected (LED goes Red and charging stops). |

## RebootReason (89)

| Value | Reason                         |
| :---- | :----------------------------- |
| 20    | Reboot                         |
| 7     | LowPowerReset                  |
| 6     | WindowWatchdogReset            |
| 5     | IndependentWindowWatchdogReset |
| 4     | SoftwareReset                  |
| 3     | BOR                            |
| 2     | PinReset                       |
| 1     | OptionByteLoaderReset          |
| 0     | FirewallReset                  |

## detectedPowerGridType (21)

| Value | detectedPowerGridType                 |
| :---- | :------------------------------------ |
| 1     | TN3Phase                              |
| 2     | TN2PhasePin234                        |
| 3     | TN1Phase                              |
| 4     | IT3Phase                              |
| 5     | IT1Phase                              |
| 30    | WarningTN2PhasePin235                 |
| 31    | WarningTN1PhaseNeutralOnPin3          |
| 32    | WARNING\_IT\_3\_PHASE\_GND\_FAULT     |
| 33    | WARNING\_IT\_1\_PHASE\_GND\_FAULT     |
| 34    | WARNING\_IT\_3\_PHASE\_GND\_FAULT\_L3 |
| 35    | WARNING\_IT\_1\_PHASE\_GND\_FAULT\_L3 |
| 36    | WARNING\_TN\_2\_PHASE\_PIN\_2\_3\_4   |
| 37    | WARNING\_TN\_3\_PHASE\_GND\_FAULT     |
| 38    | WARNING\_TN\_2\_PHASE\_GND\_FAULT     |
| 50    | ErrorNoValidPowerGridFound            |
| 51    | ErrorTN400VNeutralOnWrongPin          |
| 52    | ErrorITGroundConnectedToPin2Or3       |
|       |                                       |

<br />