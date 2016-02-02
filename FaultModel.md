# Fault model

## Case 1: Storage
### 1. Failure modes
Read
- wrong data
- old data
- data from wrong address
- fail

Write
- write wrong data
- write to wrong address
- does not write
- fail

### 2. Detection
- Write also address, checksum, versionId, statusbits
- Al errors -> fail
- Make the "decay thread" that flips status bits (error injection for testing)

### 3. Redundancy
- More copies/HD's
- All reads leads to write back on error (assumtion: not all redundant reads fail at the same time, we will be able to figure out which read was bad looking at checksum and versionId)
- "Repair thread" reads regularly

## Case 2: Messages
### Failure modes:
- Lost message (sent, not received)
- Delayed message or out of order
- Corrupted
- Duplicated
- Wrong recepient

### Detection and merging
- SessionId
- Checksum
- Ack
- Sequence number
- All errors -> lost message

### Handling with redundancy
- Timeout & resend

## Case 3: Calculations
### Failure modes:
- Does not do the next correct side effect

### Detect:
- Failed acceptance tests
- Simplify: panic/stop

### Handling with redundancy
a) Checkpoint restart
  1. Calculate
  2. Acceptance test
  3. Store
  4. Do side effect
b) Process pairs
- Two processes: Primary and backup
- Backup takes over when the primary fails
- Primary sends IAmAlive/heartbeat messages to backup
- Primary does the work
- Primary sends checkpoints to backup
c) Persistent process
- Assumes "transactional" infrastructure
- All calculations are transactions from one consistent state to another
- The processes are stateless
- Now the OS can take care of fault tolerance
