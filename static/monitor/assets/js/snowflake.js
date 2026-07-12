class SnowflakeIdGenerator {
    constructor(workerId = 1, datacenterId = 1) {
        this.workerId = workerId;this.datacenterId = datacenterId;this.sequence = 0;this.twepoch = 1288834974657n;this.workerIdBits = 5n;this.datacenterIdBits =5n;this.maxWorkerId=(1n<<this.workerIdBits)-1n;this.maxDatacenterId=(1n<<this.datacenterIdBits)-1n;this.sequenceBits=12n;this.workerIdShift=this.sequenceBits;this.datacenterIdShift=this.sequenceBits+this.workerIdBits;this.timestampLeftShift=this.sequenceBits+this.workerIdBits+this.datacenterIdBits;this.sequenceMask=(1n<<this.sequenceBits)-1n;this.lastTimestamp=-1n;
    }
    generate(){let timestamp=BigInt(Date.now());if(timestamp<this.lastTimestamp)throw new Error(`时钟回拨异常`);if(this.lastTimestamp===timestamp){this.sequence=(this.sequence+1n)&this.sequenceMask;if(this.sequence===0n)timestamp=this.tilNextMillis(this.lastTimestamp);}else this.sequence=0n;this.lastTimestamp=timestamp;const id=((timestamp-this.twepoch)<<this.timestampLeftShift)|(BigInt(this.datacenterId)<<this.datacenterIdShift)|(BigInt(this.workerId)<<this.workerIdShift)|this.sequence;return id.toString();}
    tilNextMillis(lastTimestamp){let timestamp=BigInt(Date.now());while(timestamp<=lastTimestamp)timestamp=BigInt(Date.now());return timestamp;}
}
const snowflakeGenerator = new SnowflakeIdGenerator();