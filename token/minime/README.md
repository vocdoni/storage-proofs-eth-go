# Minime storage proofs

Minime original solidity source code https://github.com/Giveth/minime/blob/master/contracts/MiniMeToken.sol

The Minime proof is composed of two storage proofs since it uses checkpoints and stores all the historical data.

So for a specific block we need to provide a checkpoint proof of a previous or equal block and a checkpoint proof of a higher block (the next one).

If the checkpoint required is the last stored, we need to provide a proof of non inclusion as a second proof.

```
     Index Slot
         │
         │
         │
┌────────┴──────────┐
│                   │
│                   │
│   Balances Map    │
│                   │
│                   │
└────────┬──────────┘
         │
   Holder Address
         │
┌────────┴──────────┐            #1       #2     #3      #4       #N
│                   │        ┌────────┬───────┬───────┬───────┬───────┐
│                   │        │ block  │ block │ block │ block │ block │
│ Checkpoints Array ├───────►├────────┼───────┼───────┼───────┼───────┤
│    ┌─────────┐    │        │ balance│balance│balance│balance│balance│
│    │  SIZE   │    │        └────────┴──┬────┴──┬────┴───────┴───────┘
└────┴─────────┴────┘                    │       │
                                         │       │
 ......................................................................
                                         │       │
                                    ┌────▼───────▼──┐
                                    │               │
                                    │     Proofs    │
                                    │               │
                                    └───────┬───────┘
                                            │
                                            ▼
                             Holder Balance on block between #2 and #3
```
