## 1. TN 2-phase row

- [x] 1.1 Test: TN 2-phase settable modes are `NL1`, `NL2`, `NL1L2`
- [x] 1.2 Test: on TN 2-phase `NL2` maps to the single-phase lock and `NL1L2` to auto
- [x] 1.3 Add the TN 2-phase row to `phaseModeMatrix`

## 2. Merge with release_3.3

- [x] 2.1 (review) Drop the `TestAdvertisedPhaseModes` case: #209 removed `AdvertisedPhaseModes`, a
  TN 2-phase charger advertises its settable set, covered by 1.1
