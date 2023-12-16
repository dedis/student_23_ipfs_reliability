import express, { urlencoded, json } from 'express'
import fetch from 'node-fetch'
const app = express()
const port = parseInt(process.env.PORT) || 3000

const HEALTH_CHECK_INTERVAL = parseInt(process.env.HEALH_INTERVAL) || 2000
const MAX_HEALTH_CHECK_TRIES = parseInt(process.env.MAX_TRIES) || 3

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

// Every 3 seconds with send a health check to the other services
// if one fails we add it to suspected list 
// each node gets 3 tries before removing from the list
setInterval(async () => {
    for (let communityIP in services) {
        // console.log(`Sending health check to ${communityIP}`)
        let success = true;
        let resp = null;
        try {
            resp = await fetch(`http://${communityIP}/health-check`)
        } catch (error) {
            success = false;
        }
        
        if (!success || resp.status != 200) {
            console.log(`Health check failed for ${communityIP}`)
            if (suspected[communityIP] == undefined) {
                suspected[communityIP] = 0
            }
            suspected[communityIP] += 1
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
    console.log(`Discovery service listening on: ${port}`)
})