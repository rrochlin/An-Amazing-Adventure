# Reference turns

Traced chat turns from real playtests, saved so they can be replayed after their traces expire (30
days) and compared across story pipelines. Each file holds one turn: its starting world, the player's
input, the model calls with their prompts and raw output, and what the server sent back, plus a
`note` saying what the turn is a test of. Who played, and on which connection, is stripped.

```
cd server
go run ./tools/replay -from testdata/turns -pipeline recorded     # offline: proves the rebuilds
go run ./tools/replay -from testdata/turns -pipeline current -o report.md   # reruns the models (needs Bedrock)
go run ./tools/replay -save testdata/turns -turn 3,5 -note "what it tests" SESSION_ID   # add turns
```

| File | Tests |
|---|---|
| `14a9a780-02` | Arrival at the meadow: baseline grounding in the authored cast |
| `14a9a780-03` | Confront Dunstan: engineer gave the player the boy's horse and doubled a coin; menu and tally in the prose |
| `14a9a780-04` | Resolver wrote reasoning and a corrected second JSON object |
| `14a9a780-05` | One turn did several steps in two places (find, return, get paid) |
| `14a9a780-09` | Hide: Tomas killed from prose; rescue opened at the wrong place |
| `a2d428dc-02` | Arrival at the meadow: a rerun invented a named child (Pip) |
| `a2d428dc-04` | Resolver marked the doll kept when only its location was known |

A candidate pipeline (the game master, #99) should get through these without repeating the failures.
