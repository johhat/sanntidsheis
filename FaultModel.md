# Fault model

##Case 1: Storage
###1. Failure modes
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

###2. Detection
- Write also address, checksum, versionId, statusbits
- Al errors -> fail
- Make the "decay thread" that flips status bits (error injection for testing)

###3. Redundancy
- More copies/HD's
- All reads leads to write back on error (assumtion: not all redundant reads fail at the same time, we will be able to figure out which read was bad looking at checksum and versionId)
- "Repair thread" reads regularly
