# v3.3.0 review: three cleanups left out of 3.3.1

The review of v3.3.0 (`main...release_3.3`) raised three items that are correct but are not bugs.
3.3.1 carries the bug fix only.

- `manager.restartClient` has one caller left, `Unregister` on an empty charger map, where it means
  "close, and undo the close if a Register raced". Its name and doc comment sit next to
  `Client.Restart`, which is the real restart. Inline it back into `Unregister` or rename it.
- `handlePhaseCurrent` and `handleOutPhase` both read `PropertyGridType` from the chargepoint props
  with the same lookup, type assertion and comment. One helper would hold it.
- `InCurrentT2` is supported for every charger, so on TN the neutral current reaches
  `handlePhaseCurrent`, which looks the chargepoint service up before it can drop the reading. One
  lookup per T2 observation; a grid type kept in the handler would let it return first.

Open: take them into the next release?
