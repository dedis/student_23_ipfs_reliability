import express, { urlencoded, json } from 'express'
import fetch from 'node-fetch'
import fs from 'fs/promises'
import { mkdirp } from 'mkdirp'
const app = express()
const port = parseInt(process.env.PORT) || 3000

const HEALTH_CHECK_INTERVAL = parseInt(process.env.HEALH_INTERVAL) || 2000
const MAX_HEALTH_CHECK_TRIES = parseInt(process.env.MAX_TRIES) || 3

const DEPTH = parseInt(process.env.DEPTH) || 3
const REPLICATION_FACTOR = parseInt(process.env.REPLICATION_FACTOR) || 3
const TOTAL_PEERS = parseInt(process.env.TOTAL_PEERS) || 10
const FAILED_PEERS = parseInt(process.env.FAILED_PEERS) || 3
const REPAIR_PEERS = parseInt(process.env.REPAIR_PEERS) || 3
const FILE_SIZE = process.env.FILE_SIZE || "25MB"

var done = false;

var services = {
    'exampleCommunityIpWithPort' : { //for example this will be community0:7070
        'clusterIP': 'example-cluster-ip',
        'cluserPort': 3000,
        'ipfsIP': 'example-ipfs-ip',
        'ipfsPort': 5001,
    }
}

var clusterToCommunity = {
    'example-cluster-ip': 'exampleCommunityIpWithPort'
}

var suspected = {
    'exampleCommunityIpWithPort': 0
}

app.use(urlencoded({ extended: true }));
app.use(json());


app.post('/announce', (req, res) => {
    let communityIP = req.body.communityIP
    let clusterIP = req.body.clusterIP
    let clusterPort = req.body.clusterPort
    let ipfsIP = req.body.ipfsIP
    let ipfsPort = req.body.ipfsPort

    services[communityIP] = {
        'clusterIP': clusterIP,
        'clusterPort': clusterPort,
        'ipfsIP': ipfsIP,
        'ipfsPort': ipfsPort,
    }

    clusterToCommunity[clusterIP] = communityIP
    res.sendStatus(200)

    console.log(`Announced ${communityIP} with clusterIP ${clusterIP} and clusterPort ${clusterPort} and ipfsIP ${ipfsIP} and ipfsPort ${ipfsPort}`)
})


app.get('/cluster-to-community', (req, res) => {
    res.send(clusterToCommunity[req.query.clusterIP])
})

app.get('/peers', (req, res) => {
    res.json(services)
})

app.get('/done', (req, res) => { 
    res.json(done)
})

app.post('/reportMetrics', async (req, res) => {
    // persist json body to file 
    // the file will be stored 

    let data = req.body
    let output = {
        'depth': DEPTH,
        'replicationFactor': REPLICATION_FACTOR,
        'totalPeers': TOTAL_PEERS,
        'failedPeers': FAILED_PEERS,
        'repairPeers': REPAIR_PEERS,
        'metrics': data
    }

    res.status(200).send('OK')


    // write output to /data/DEPTH_REPLICATIONFACTOR_TOTALPEERS_FAILEDPEERS_REPAIRPEERS/collab_repair_<current datetime>.json
    let date = new Date()
    let dir_name = `/data/${DEPTH}_${REPLICATION_FACTOR}_${TOTAL_PEERS}_${FAILED_PEERS}_${REPAIR_PEERS}_${FILE_SIZE}`
    let filename = `${dir_name}/collab_repair_${date.getFullYear()}_${date.getMonth()}_${date.getDate()}_${date.getHours()}_${date.getMinutes()}_${date.getSeconds()}.json`
    await mkdirp(dir_name)
    await fs.writeFile(filename, JSON.stringify(output))

    done = true;
    console.log(`Metrics written to ${filename}`)
    
})


app.post('/reportDownloadMetrics', async (req, res) => {
    // persist json body to file 
    // the file will be stored 

    let data = req.body
    let output = {
        'depth': DEPTH,
        'replicationFactor': REPLICATION_FACTOR,
        'totalPeers': TOTAL_PEERS,
        'failedPeers': FAILED_PEERS,
        'repairPeers': REPAIR_PEERS,
        'metrics': data
    }

    res.status(200).send('OK')


    // write output to /data/DEPTH_REPLICATIONFACTOR_TOTALPEERS_FAILEDPEERS_REPAIRPEERS/single_repair_<current datetime>.json
    let date = new Date()
    let dir_name = `/data/${DEPTH}_${REPLICATION_FACTOR}_${TOTAL_PEERS}_${FAILED_PEERS}_${REPAIR_PEERS}_${FILE_SIZE}`
    let filename = `${dir_name}/single_repair_${date.getFullYear()}_${date.getMonth()}_${date.getDate()}_${date.getHours()}_${date.getMinutes()}_${date.getSeconds()}.json`
    await mkdirp(dir_name)
    await fs.writeFile(filename, JSON.stringify(output))

    done = true;
    console.log(`Metrics written to ${filename}`)
    
})
// Every 3 seconds with send a health check to the other services
// if one fails we add it to suspected list 
// each node gets 3 tries before removing from the list
setInterval(async () => {
    // console.log(`Running health check with ${Object.keys(services).length} services`)
    for (let communityIP in services) {
        // console.log(`Sending health check to ${communityIP}`)
        let success = true;
        let resp = null;
        try {
            resp = await fetch(`http://${communityIP}/health-check`)
        } catch (error) {
            success = false;
        }
        
        if (services[communityIP] == undefined) {
            continue
        }
        if (!success || resp.status != 200) {
            console.log(`Health check failed for ${communityIP}`)
            if (suspected[communityIP] == undefined) {
                suspected[communityIP] = 0
            }
            suspected[communityIP] += 1
            if (services[communityIP] == undefined) {
                continue
            }
            if (suspected[communityIP] == MAX_HEALTH_CHECK_TRIES) {
                console.log(`Removing ${communityIP} from services`)
                delete clusterToCommunity[services[communityIP].clusterIP]
                delete services[communityIP]
                delete suspected[communityIP]
            }
        } else {
            // console.log(`Health check passed for ${communityIP}`)
            suspected[communityIP] = 0
        }

    }
}, HEALTH_CHECK_INTERVAL)



app.listen(port, () => {
    console.log(`Discovery service listening on: ${port} with ${HEALTH_CHECK_INTERVAL} interval and ${MAX_HEALTH_CHECK_TRIES} tries`)
})