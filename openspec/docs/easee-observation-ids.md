---
updatedAt: 2026-10-16T11:41:29.000Z
agentTools:
  projectIndex: https://developer.easee.com/llms.txt
---

# Observation Ids

Here you will find a complete list of observation ids that are available from our Observations endpoint

<HTMLBlock>{`
</header>
<div class="page-body">
<table class="collection-content">
<thead>
<tr>
<th>ID</th>
<th>Observation</th>
<th>Info</th>
<th>Value</th>
<th>Type</th>
<th>Response</th>
</tr>
</thead>
<tbody>
<tr id="1d338f95-5e13-4944-af76-f6157bd4a0fa">
<td class="cell-nYKS">1</td>
<td class="cell-title">SELF TEST RESULT</td>
<td class="cell-zJgg">PASSED or error codes </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4355bd38-6aa6-4f79-b118-ef86fb7054e7">
<td class="cell-nYKS">2</td>
<td class="cell-title">SELF TEST DETAILS</td>
<td class="cell-zJgg">JSON with details from self-tes </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg"> JSON</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4355bd38-6aa6-4f79-b118-ef86fb7054e7">
<td class="cell-nYKS">10</td>
<td class="cell-title">WIFI EVENT  </td>
<td class="cell-zJgg">Wifi event </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event </td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="4dd46b8d-f69a-459b-bf0f-81c0e8e9743a">
<td class="cell-nYKS">11</td>
<td class="cell-title">CHARGER OFFLINE REASON</td>
<td class="cell-zJgg">Enum describing why charger is offline </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="60abe6d4-94a2-40ca-9cb1-385f15215f24">
<td class="cell-nYKS">15</td>
<td class="cell-title">LOCAL PRE AUTHORIZE ENABLED</td>
<td class="cell-zJgg">Preauthorize with whitelist enabled. Readback on setting</td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="2b9c8ff5-b05f-4482-b455-83e2069f4fa4">
<td class="cell-nYKS">16</td>
<td class="cell-title">LOCAL AUTHORIZE OFFLINE ENABLED</td>
<td class="cell-zJgg">Allow offline charging for whitelisted RFID token. Readback on setting</td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="496bfbf6-318a-410f-94eb-2cad6df36e41">
<td class="cell-nYKS">17</td>
<td class="cell-title">ALLOW OFFLINE TX FOR UNKNOWN ID</td>
<td class="cell-zJgg">Allow offline charging for all RFID tokens. Readback on setting</td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">18</td>
<td class="cell-title">ERRATIC EVMAX TOGGLES</td>
<td class="cell-zJgg">0 == erratic checking disabled, otherwise the number of toggles between states Charging and Charging Complate that will trigger an error </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">19</td>
<td class="cell-title">BACKPLATE TYPE</td>
<td class="cell-zJgg">erratic checking type of backplate, type =0 corrsponds to comercialcharger, type =1 corrsponds to testing charger </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">20</td>
<td class="cell-title">SITE STRUCTURE</td>
<td class="cell-zJgg"> checking site structure and displaying address,conctac info, installer, owner etc </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">JSON</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4211c41c-dc9b-417d-a1da-54f84cb8e64b">
<td class="cell-nYKS">21</td>
<td class="cell-title">DETECTED POWER GRID TYPE</td>
<td class="cell-zJgg">Detected power grid type according to PowerGridType table </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">22</td>
<td class="cell-title">Circuit Max Current P1</td>
<td class="cell-zJgg">Circuit max current in Ampers of Phase 1</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">23</td>
<td class="cell-title">Circuit Max Current P2</td>
<td class="cell-zJgg">Circuit max current in Ampers of Phase 2</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">24</td>
<td class="cell-title">Circuit Max Current P3</td>
<td class="cell-zJgg">Circuit max current in Ampers of Phase 3</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">26</td>
<td class="cell-title">Site ID String</td>
<td class="cell-zJgg">Site ID as a string </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">27</td>
<td class="cell-title">Site ID Numeric</td>
<td class="cell-zJgg">Site ID as a numeric </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0e7ec78e-9190-46b9-abd4-5a40b161f397">
<td class="cell-nYKS">28</td>
<td class="cell-title">RFID TIMEOUT AUTH</td>
<td class="cell-zJgg">Timeout set for authentication for when Authentication == True </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="8c925361-fad8-4129-a05b-d47f90bacc55">
<td class="cell-nYKS">30</td>
<td class="cell-title">LOCK CABLE PERMANENTLY</td>
<td class="cell-zJgg">Lock type2 cable permanently </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="8f19b3e8-830e-4641-879a-c504e808162e">
<td class="cell-nYKS">31</td>
<td class="cell-title">IS ENABLED</td>
<td class="cell-zJgg">Set true to enable charger, false disables charger </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="8f19b3e8-830e-4641-879a-c504e808162e">
<td class="cell-nYKS">32</td>
<td class="cell-title">TEMPERATURE MONITOR STATE</td>
<td class="cell-zJgg">Retrieves temperature of the charger </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
<tr id="1ef32dc7-3521-41a2-91ea-ce556dd208db">
</tr>
<tr id="8f19b3e8-830e-4641-879a-c504e808162e">
<td class="cell-nYKS">36</td>
<td class="cell-title"> WIFI SSID</td>
<td class="cell-zJgg"> Network name charger/s are connected to </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
<tr id="1ef32dc7-3521-41a2-91ea-ce556dd208db">
</tr>
<tr id="8f19b3e8-830e-4641-879a-c504e808162e">
<td class="cell-nYKS">38</td>
<td class="cell-title">PHASE MODE</td>
<td class="cell-zJgg">erratic showing what phase mode is the charger connected to, phase mode ==1 --> one phase, phase mode==3 --> 3 phase </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
<tr id="1ef32dc7-3521-41a2-91ea-ce556dd208db">
</tr>
<tr id="8f19b3e8-830e-4641-879a-c504e808162e">
<td class="cell-nYKS">40</td>
<td class="cell-title">LED STRIP BRIGHTNESS</td>
<td class="cell-zJgg">erratic showing in integer the LED brightness </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot </td>
<td class="cell-zJgg"> </td>
<tr id="1ef32dc7-3521-41a2-91ea-ce556dd208db">
</tr>
<td class="cell-nYKS">41</td>
<td class="cell-title">LOCAL AUTHORIZATION REQUIRED</td>
<td class="cell-zJgg">Local RFID authorization is required for charging </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">User Options</td>
</tr>
<tr id="a1b23e62-e773-4d53-afb2-ebda1e0f9f3e">
<td class="cell-nYKS">42</td>
<td class="cell-title">AUTHORIZATION REQUIRED</td>
<td class="cell-zJgg">Authorization is required for charging </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="32703dcf-8df7-4e54-baa1-25e04817b581">
<td class="cell-nYKS">43</td>
<td class="cell-title">REMOTE START REQUIRED</td>
<td class="cell-zJgg">Remote start required flag</td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="be75f50d-19f3-4c3c-9b0e-b36ffa5e0312">
<td class="cell-nYKS">44</td>
<td class="cell-title">SMART BUTTON ENABLED</td>
<td class="cell-zJgg">Smart button is enabled</td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="7368faa9-0dae-43ea-b022-4ab0eeacfb00">
<td class="cell-nYKS">45</td>
<td class="cell-title">OFFLINE CHARGING MODE</td>
<td class="cell-zJgg">Charger behaviour when offline</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="6da856a5-633d-4786-a229-ee6a3e9ff480">
<td class="cell-nYKS">46</td>
<td class="cell-title">LEDMODE</td>
<td class="cell-zJgg">Charger LED mode</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="38b56627-445f-4840-8970-5ff07730fed1">
<td class="cell-nYKS">47</td>
<td class="cell-title">MAX CHARGER CURRENT</td>
<td class="cell-zJgg">Max current this charger is allowed to offer to car (A). Non volatile. </td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="80491232-5bf4-43ac-8963-504de7b29703">
<td class="cell-nYKS">48</td>
<td class="cell-title">DYNAMIC CHARGER CURRENT</td>
<td class="cell-zJgg">Max current this charger is allowed to offer to car (A). Volatile </td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="b831ed90-36e1-4516-a9b4-48b6dc70f2f7">
<td class="cell-nYKS">50</td>
<td class="cell-title">MAX CURRENT OFFLINE FALLBACK P1</td>
<td class="cell-zJgg">Maximum circuit current P1 when offline </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="6b99d540-5f25-44d3-9c6c-5562bbc7fa9f">
<td class="cell-nYKS">51</td>
<td class="cell-title">MAX CURRENT OFFLINE FALLBACK P2</td>
<td class="cell-zJgg">Maximum circuit current P2 when offline </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d00cad7e-4b50-4e1d-bde3-64be9f935529">
<td class="cell-nYKS">52</td>
<td class="cell-title">MAX CURRENT OFFLINE FALLBACK P3</td>
<td class="cell-zJgg">Maximum circuit current P3 when offline </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d00cad7e-4b50-4e1d-bde3-64be9f935529">
<td class="cell-nYKS">54</td>
<td class="cell-title">RELEASE CABLE AT POWER OFF</td>
<td class="cell-zJgg">If set to Yes, cable is released when power off. If set to No, cable not released </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d1c14958-2695-44a3-bb8d-aee22aeacb45">
<td class="cell-nYKS">62</td>
<td class="cell-title">CHARGING SCHEDULE</td>
<td class="cell-zJgg">Charging schedule </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Json</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="2ab4b164-bbce-449e-a4fb-224698480c1b">
<td class="cell-nYKS">65</td>
<td class="cell-title">PAIRED EQUALIZER</td>
<td class="cell-zJgg">Paired equalizer details </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">69</td>
<td class="cell-title">PAIRED USER IDTOKEN</td>
<td class="cell-zJgg">Observed user token when charger put in RFID pairing mode</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">70</td>
<td class="cell-title">Circuit Total Allocated Phase Conductor Current L1</td>
<td class="cell-zJgg">Circuit Total Allocated Phase Conductor Current L1</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">71</td>
<td class="cell-title">Circuit Total Allocated Phase Conductor Current L2</td>
<td class="cell-zJgg">Circuit Total Allocated Phase Conductor Current L2</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">72</td>
<td class="cell-title">Circuit Total Allocated Phase Conductor Current L3</td>
<td class="cell-zJgg">Circuit Total Allocated Phase Conductor Current L3</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">73</td>
<td class="cell-title">Circuit Total Phase Conductor Current L1</td>
<td class="cell-zJgg">Circuit Total Phase Conductor Current L1</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">74</td>
<td class="cell-title">Circuit Total Phase Conductor Current L2</td>
<td class="cell-zJgg">Circuit Total Phase Conductor Current L2</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="06a6a4dc-cb9c-492a-b324-dcf7463d7588">
<td class="cell-nYKS">75</td>
<td class="cell-title">Circuit Total Phase Conductor Current L3</td>
<td class="cell-zJgg">Circuit Total Phase Conductor Current L3</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="a76dce8a-ee75-4d3c-956e-15a670014297">
<td class="cell-nYKS">76</td>
<td class="cell-title">NUMBER OF CARS CONNECTED</td>
<td class="cell-zJgg">Number of cars connected to this circuit</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="c10ea66c-9e8f-4128-ab06-5a354d74352c">
<td class="cell-nYKS">77</td>
<td class="cell-title">NUMBER OF CARS CHARGING</td>
<td class="cell-zJgg">Number of cars currently charging </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d78cc0b3-7672-435b-aaae-ddaaeb30f903">
<td class="cell-nYKS">78</td>
<td class="cell-title">NUMBER OF CARS IN QUEUE</td>
<td class="cell-zJgg">Number of cars currently in queue, waiting to be allocated power </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4b70b44f-5d57-4c27-afe9-92eb02f18389">
<td class="cell-nYKS">79</td>
<td class="cell-title">NUMBER OF CARS FULLY CHARGED</td>
<td class="cell-zJgg">Number of cars that appear to be fully charged </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4b70b44f-5d57-4c27-afe9-92eb02f18389">
<td class="cell-nYKS">80</td>
<td class="cell-title">SOFTWARE RELEASE </td>
<td class="cell-zJgg">Firmware of charger </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4b70b44f-5d57-4c27-afe9-92eb02f18389">
<td class="cell-nYKS">81</td>
<td class="cell-title">ICCID</td>
<td class="cell-zJgg">ICCID number  </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4b70b44f-5d57-4c27-afe9-92eb02f18389">
<td class="cell-nYKS">84</td>
<td class="cell-title">Mobile Network Operator</td>
<td class="cell-zJgg">Name of mobile network operator</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="55c49024-ed2a-4aab-9960-65f6257b0c23">
<td class="cell-nYKS">89</td>
<td class="cell-title">REBOOT REASON</td>
<td class="cell-zJgg">Reason of reboot. Bitmask of flags.</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="3f95182e-ee6f-4409-a640-9fadea39e652">
<td class="cell-nYKS">96</td>
<td class="cell-title">REASON FOR NO CURRENT</td>
<td class="cell-zJgg">Enum describing why a charger with a car connected is not offering current to the car </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="5523038b-ec8c-4d74-b949-30db8723f27a">
<td class="cell-nYKS">97</td>
<td class="cell-title">LOAD BALANCING NUMBER OF CONNECTED CHARGERS</td>
<td class="cell-zJgg">Number of connected chargers in the load balancing. Including the master. Sent from Master only. </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d0b7032b-d84d-4f46-9e38-11f1830f7343">
<td class="cell-nYKS">98</td>
<td class="cell-title">UDP NUM OF CONNECTED NODES</td>
<td class="cell-zJgg">Number of chargers connected to master through UDP and WIFI </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="55c49024-ed2a-4aab-9960-65f6257b0c23">
<td class="cell-nYKS">99</td>
<td class="cell-title">Local Connection</td>
<td class="cell-zJgg">Secondarychargers only. Current connection to master, 0 = None, 1= Radio, 2 = WIFI UDP, 3 = Radio and WIFI UDP.</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Boot </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="99923c02-8d8d-4e3d-b551-d0dfb0f63d20">
<td class="cell-nYKS">100</td>
<td class="cell-title">PILOT MODE</td>
<td class="cell-zJgg">Pilot Mode Letter (A-F) </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="d0c0fc95-afd1-4538-b0ed-15781f8f54ca">
<td class="cell-nYKS">102</td>
<td class="cell-title">SMART CHARGING</td>
<td class="cell-zJgg">Smart charging state enabled by capacitive touch button </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="861e837e-f831-4b98-99dc-df7018644278">
<td class="cell-nYKS">103</td>
<td class="cell-title">CABLE LOCKED</td>
<td class="cell-zJgg">Cable lock state </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="f17dc28e-002a-4fb6-9d84-866e263decb9">
<td class="cell-nYKS">104</td>
<td class="cell-title">CABLE RATING</td>
<td class="cell-zJgg">Cable rating read (Amperes)</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="f17dc28e-002a-4fb6-9d84-866e263decb9">
<td class="cell-nYKS">107</td>
<td class="cell-title">BACKPLATE ID </td>
<td class="cell-zJgg">Backplate id</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="fe9b2d94-2a7e-4110-a1c7-b1ec55f82d30">
<td class="cell-nYKS">108</td>
<td class="cell-title">USER IDTOKEN REVERSED</td>
<td class="cell-zJgg">User ID token string from RFID reading (NB! Must reverse these strings) </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="3e64c16b-e98e-4446-8265-dfb0ba9dc294">
<td class="cell-nYKS">109</td>
<td class="cell-title">CHARGER OP MODE</td>
<td class="cell-zJgg">Charger operation mode according to charger mode table </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">110</td>
<td class="cell-title">OUTPUT PHASE</td>
<td class="cell-zJgg">Active output phase(s) to EV according to output phase type table. </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">111</td>
<td class="cell-title">Dynamic Circuit Current P1 </td>
<td class="cell-zJgg">Dynamic Circuit current in P1 in Amps </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">112</td>
<td class="cell-title">Dynamic Circuit Current P2</td>
<td class="cell-zJgg">ADynamic Circuit current in P2 in Amps </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">113</td>
<td class="cell-title">Dynamic Circuit Current P3</td>
<td class="cell-zJgg">Dynamic Circuit current in P3 in Amps </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">114</td>
<td class="cell-title">Output Current</td>
<td class="cell-zJgg">Available current signaled to car with pilot tone in Amps </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">115</td>
<td class="cell-title">Derated Current</td>
<td class="cell-zJgg">Available current after derating </td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="46ddcb7c-af13-4cd6-881c-78e55086261a">
<td class="cell-nYKS">116</td>
<td class="cell-title">DERATING ACTIVE</td>
<td class="cell-zJgg">Available current is limited by the charger due to high temperature </td>
<td class="cell-zJgg">Boolean</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0ceea040-28d0-453e-b454-4bea069f875c">
<td class="cell-nYKS">118</td>
<td class="cell-title">Error String </td>
<td class="cell-zJgg">Descriptive Error String  </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="811af5c2-b285-4d59-8559-01ba85ba8062">
<td class="cell-nYKS">119</td>
<td class="cell-title">ERROR CODE</td>
<td class="cell-zJgg">Error code according to error code table [event] [Integer] </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="61c6a480-ee60-4062-bf8f-de3514257480">
<td class="cell-nYKS">120</td>
<td class="cell-title">TOTAL POWER</td>
<td class="cell-zJgg">Total power (kW)</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="20ee2587-7127-4e6f-89ec-2522af95f4ea">
<td class="cell-nYKS">121</td>
<td class="cell-title">SESSION ENERGY</td>
<td class="cell-zJgg">Session accumulated energy (kWh) </td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="4142da5d-f592-45d6-b853-10806c3ce54d">
<td class="cell-nYKS">122</td>
<td class="cell-title">ENERGY PER HOUR</td>
<td class="cell-zJgg">Accumulated energy per hour (kWh)</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="382cc23b-3517-4750-9c70-87328da0f5a3">
<td class="cell-nYKS">123</td>
<td class="cell-title">LEGACY EV STATUS </td>
<td class="cell-zJgg"> 0 = not legacy ev, 1 = legacy ev detected, 2 = reviving ev</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="382cc23b-3517-4750-9c70-87328da0f5a3">
<td class="cell-nYKS">124</td>
<td class="cell-title">LIFETIME ENERGY</td>
<td class="cell-zJgg">Accumulated energy in the lifetime of the charger (kWh)</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="382cc23b-3517-4750-9c70-87328da0f5a3">
<td class="cell-nYKS">125</td>
<td class="cell-title">LIFETIME RELAY SWITCHES</td>
<td class="cell-zJgg">Total number of relay switches in the lifetime of the charger (irrespective of the number of phases used)</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="382cc23b-3517-4750-9c70-87328da0f5a3">
<td class="cell-nYKS">126</td>
<td class="cell-title">LIFETIME HOURS</td>
<td class="cell-zJgg">Total number of hours in operation </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg"> </td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="898bc591-00b3-4f58-b498-b98dda6cb6a5">
<td class="cell-nYKS">128</td>
<td class="cell-title">USER IDTOKEN</td>
<td class="cell-zJgg">User ID token string from RFID reading </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="baba4f07-1c97-4a5d-a606-502665e4bd3d">
<td class="cell-nYKS">129</td>
<td class="cell-title">CHARGING SESSION</td>
<td class="cell-zJgg">Charging sessions</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Json</td>
</tr>
<tr id="a607cf47-15ad-4a30-b657-df4006c4c688">
<td class="cell-nYKS">130</td>
<td class="cell-title">CELL RSSI</td>
<td class="cell-zJgg">Cellular signal strength (dBm)</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="babce91c-6f7d-48b6-9ee6-88c3894d469a">
<td class="cell-nYKS">131</td>
<td class="cell-title">CELL RAT</td>
<td class="cell-zJgg">Cellular radio access technology according to RAT table</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="9a9ab530-37cc-4f8b-bd9f-bca2b61d56c1">
<td class="cell-nYKS">132</td>
<td class="cell-title">WI FI RSSI</td>
<td class="cell-zJgg">WiFi signal strength (dBm)</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="9a9ab530-37cc-4f8b-bd9f-bca2b61d56c1">
<td class="cell-nYKS">133</td>
<td class="cell-title">CELL ADDRESS</td>
<td class="cell-zJgg">IP address assigned by cellular network</td>
<td class="cell-zJgg">Sring</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="9a9ab530-37cc-4f8b-bd9f-bca2b61d56c1">
<td class="cell-nYKS">134</td>
<td class="cell-title">WIFI ADDRESS</td>
<td class="cell-zJgg">IP address assigned by WiFi network</td>
<td class="cell-zJgg">Sring</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
<tr id="f4a780c1-6699-4fa2-ba2f-7116382af404">
<td class="cell-nYKS">136</td>
<td class="cell-title">LOCAL RSSI</td>
<td class="cell-zJgg">Local radio signal strength (dBm)</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="0fac6b93-b737-4e63-b586-e849b98437f7">
<td class="cell-nYKS">138</td>
<td class="cell-title">LOCAL TX POWER</td>
<td class="cell-zJgg">Local radio transmission power (dBm) </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Telemetry</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">139</td>
<td class="cell-title">LOCAL STATE</td>
<td class="cell-zJgg">Local radio state </td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">141</td>
<td class="cell-title">CURRENT CONNECTION</td>
<td class="cell-zJgg">Radio access technology in use: 0 = cellular, 1 = wifi</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">146</td>
<td class="cell-title">LOCAL NODE TYPE</td>
<td class="cell-zJgg">erratic=0-Unconfigured, 1-Master, 2-Extender, 3-End devicet</td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">  </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">149</td>
<td class="cell-title">LOCAL PARENT ADDR OR NUM OF NODES</td>
<td class="cell-zJgg">If master-Number of secondary connected, If secondary- Address parent </td>
<td class="cell-zJgg">Integer</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg">Easee Internal usage </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">150</td>
<td class="cell-title">TEMP MAX</td>
<td class="cell-zJgg">Maximum temperature for all sensors</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<tr id="ce54c29e-a3a3-4978-93e9-eef14f7191ab">
<td class="cell-nYKS">152</td>
<td class="cell-title">TEMP INPUT T2</td>
<td class="cell-zJgg">Temperature at input T2</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">153</td>
<td class="cell-title">TEMP INPUT T3</td>
<td class="cell-zJgg">Temperature at input T3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">154</td>
<td class="cell-title">TEMP INPUT T4</td>
<td class="cell-zJgg">Temperature at input T4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">155</td>
<td class="cell-title">TEMP INPUT T5</td>
<td class="cell-zJgg">Temperature at input T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">160</td>
<td class="cell-title">TEMP OUTPUT N</td>
<td class="cell-zJgg">Temperature at type 2 connector plug for N</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">161</td>
<td class="cell-title">TEMP OUTPUT L1</td>
<td class="cell-zJgg">Temperature at type 2 connector plug for L1</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">162</td>
<td class="cell-title">TEMP OUTPUT L2</td>
<td class="cell-zJgg">Temperature at type 2 connector plug for L2</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">163</td>
<td class="cell-title">TEMP OUTPUT L3</td>
<td class="cell-zJgg">Temperature at type 2 connector plug for L3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">170</td>
<td class="cell-title">TEMP AMBIENT</td>
<td class="cell-zJgg">Ambient temperature [Celsius]</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">172</td>
<td class="cell-title">INT REL HUMIDITY </td>
<td class="cell-zJgg">Internal relative humidity </td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">173</td>
<td class="cell-title">BACKPLATE LOCKED </td>
<td class="cell-zJgg">Back plate confirmed locked</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">182</td>
<td class="cell-title">INT CURRENT T2 </td>
<td class="cell-zJgg">Calculated current RMS for input T2</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">183</td>
<td class="cell-title">INT CURRENT T3</td>
<td class="cell-zJgg">Calculated current RMS for input T3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">184</td>
<td class="cell-title">INT CURRENT T4</td>
<td class="cell-zJgg">Calculated current RMS for input T4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">185</td>
<td class="cell-title">INT CURRENT T5</td>
<td class="cell-zJgg">Calculated current RMS for input T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">190</td>
<td class="cell-title">IN VOLT T1T2 </td>
<td class="cell-zJgg">Input voltage RMS between T1 and T2</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">191</td>
<td class="cell-title">IN VOLT T1T3 </td>
<td class="cell-zJgg">Input voltage RMS between T1 and T3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">192</td>
<td class="cell-title">IN VOLT T1T4</td>
<td class="cell-zJgg">Input voltage RMS between T1 and T4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">193</td>
<td class="cell-title">IN VOLT T1T5</td>
<td class="cell-zJgg">Input voltage RMS between T1 and T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">194</td>
<td class="cell-title">IN VOLT T2T3 </td>
<td class="cell-zJgg">Input voltage RMS between T2 and T3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">195</td>
<td class="cell-title">IN VOLT T2T4 </td>
<td class="cell-zJgg">Input voltage RMS between T2 and T4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">196</td>
<td class="cell-title">IN VOLT T2T5 </td>
<td class="cell-zJgg">Input voltage RMS between T2 and T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">197</td>
<td class="cell-title">IN VOLT T3T4</td>
<td class="cell-zJgg">Input voltage RMS between T3 and T4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">198</td>
<td class="cell-title">IN VOLT T3T5</td>
<td class="cell-zJgg">Input voltage RMS between T3 and T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">199</td>
<td class="cell-title">IN VOLT T4T5</td>
<td class="cell-zJgg">Input voltage RMS between T4 and T5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">202</td>
<td class="cell-title">OUT VOLT PIN 1&2</td>
<td class="cell-zJgg">Output voltage RMS between type 2 pin 1 and 2</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">203</td>
<td class="cell-title">OUT VOLT PIN 1&3</td>
<td class="cell-zJgg">Output voltage RMS between type 2 pin 1 and 3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">204</td>
<td class="cell-title">OUT VOLT PIN 1&4</td>
<td class="cell-zJgg">Output voltage RMS between type 2 pin 1 and 4</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">205</td>
<td class="cell-title">OUT VOLT PIN 1&5</td>
<td class="cell-zJgg">Output voltage RMS between type 2 pin 1 and 5</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">206</td>
<td class="cell-title">OUT VOLT PIN 2&3</td>
<td class="cell-zJgg">Output voltage RMS between type 2 pin 2 and 3</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg">Event</td>
<td class="cell-zJgg"> </td>
</tr>
<td class="cell-nYKS">223</td>
<td class="cell-title">CHARGE SESSION START</td>
<td class="cell-zJgg">Last session details</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">JSON</td>
<td class="cell-zJgg"></td>
</tr>
<td class="cell-nYKS">230</td>
<td class="cell-title">EQ AVAILABLE CURRENT P1</td>
<td class="cell-zJgg">Available current for charging on P1 according to Equalizer</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"></td>
<td class="cell-zJgg"></td>
</tr>
<td class="cell-nYKS">231</td>
<td class="cell-title">EQ AVAILABLE CURRENT P2</td>
<td class="cell-zJgg">Available current for charging on P2 according to Equalizer</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"></td>
<td class="cell-zJgg"></td>
</tr>
<td class="cell-nYKS">232</td>
<td class="cell-title">EQ AVAILABLE CURRENT P3</td>
<td class="cell-zJgg">Available current for charging on P3 according to Equalizer</td>
<td class="cell-zJgg">Double</td>
<td class="cell-zJgg"></td>
<td class="cell-zJgg"></td>
</tr>
<td class="cell-nYKS">250</td>
<td class="cell-title">CONNECTED TO CLOUD</td>
<td class="cell-zJgg">If charger connected to cloud or not</td>
<td class="cell-zJgg">BOOLEAN</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg"></td>
</tr>

<td class="cell-nYKS">251</td>
<td class="cell-title">CLOUD DISCONNECT REASON</td>
<td class="cell-zJgg">Reason why charger disconnected from cloud</td>
<td class="cell-zJgg">String</td>
<td class="cell-zJgg">Boot</td>
<td class="cell-zJgg">Easee Internal usage</td>
</tr>
</tbody>
</table>
</div>
</article>
`}</HTMLBlock>