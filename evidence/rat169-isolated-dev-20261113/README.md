# PR #169 — isolated DEV Thing-type selector acceptance (13 November 2026)

Source: `9ea1502f4bb93a2284dc29864aeed1cb8057408c`. PR remains **open / draft / unmerged**. No redeployment, repeat migration, reset, seed, PR merge, or LIVE change performed.

## Results

- Deployed Playwright acceptance: **PASS**, Workbench receipt `rat169-acceptance-20261113-browser-a01`, including 14 checks, 8 screenshots captured, seven copied here for visual review.
- Served `/VERSION` and canonical API health: `rat-candidate+9ea1502f4bb93a2284dc29864aeed1cb8057408c`.
- Running API pod digest: `sha256:3f313da265f9b19c166073fa6642922cf8ee2e897318b22ce747826a841395f3`.
- Running RAT frontend pod digest: `sha256:9e9feff67611009f1f9193345754fd929cd9dd1449e6d3b30ae3a5486c085638`.
- Canonical post-migration DB: recorded, mapping column present, 12/12 business-location keys correct and creator_collective→organisation correct, digital_resource included in persisted general rating scheme.
- Canonical persistence: test classification ID and key both present, digital_resource structural kind, one primary entity_classifications assignment in unverified state, one rating event, Thing published after initial rating, journey present. Deep readback `rat169-acceptance-20261113-db-deep-d03`.
- Privacy/catalogue: DJI and CJ Investigates served by canonical DEV with private-RUM-persons-excluded marker. Owner’s manual test case not used.

## Real browser screenshots

- [01-no-match](01-no-match.jpg)
- [03-lookup-failure-preserved](03-lookup-failure-preserved.jpg)
- [04-creation-failure-preserved](04-creation-failure-preserved.jpg)
- [05-created-type-preserved](05-created-type-preserved.jpg)
- [06-initial-rating-pending](06-initial-rating-pending.jpg)
- [08-fresh-form-reuse](08-fresh-form-reuse.jpg)
- [07-initial-rating-complete](07-initial-rating-complete.jpg)

## Notes

The first SQL probe (`rat169-acceptance-20261113-db-proof-c01`) transiently failed to locate the classification/association while finding the published Thing/rating. The deeper, independently rerun database query (`rat169-acceptance-20261113-db-deep-d03`) found the classification by both ID and key, primary assignment and correct mapping. Both raw diagnostic results are preserved in `acceptance-report.json`; the original failure is not concealed.

This branch holds **acceptance evidence only**. No RUM source or runtime mutations are included. The test created separate, non-person isolated DEV data through the intended UI.
