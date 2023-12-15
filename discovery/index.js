const express = require('express')
const fetch = require('node-fetch')
const app = express()
const port = process.env.PORT || 3000

const HEALTH_CHECK_INTERVAL = 3000
const MAX_HEALTH_CHECK_TRIES = 3

services = {
    'exampleCommunityIpWithPort' : { //for example this will be community0:7070
        'clusterIP': 'example-cluster-ip',
        'cluserPort': 3000,
        'ipfsIP': 'example-ipfs-ip',
        'ipfsPort': 5001,
    }
}

clusterToCommunity = {
    'example-cluster-ip': 'exampleCommunityIpWithPort'
}

suspected = {
    'exampleCommunityIpWithPort': 0
}

app.use(express.urlencoded({ extended: true }));
app.use(express.json());


app.post('/announce', (req, res) => {
    communityIP = req.body.communityIP
    clusterIP = req.body.clusterIP
    clusterPort = req.body.clusterPort
    ipfsIP = req.body.ipfsIP
    ipfsPort = req.body.ipfsPort

    services[communityIP] = {
        'clusterIP': clusterIP,
        'clusterPort': clusterPort,
        'ipfsIP': ipfsIP,
        'ipfsPort': ipfsPort,
    }

    clusterToCommunity[clusterIP] = communityIP
    res.sendStatus(200)
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
    for (communityIP in services) {
        const resp = await fetch(`http://${communityIP}/health-check`)
        if (resp.status != 200) {
            if (suspected[communityIP] == undefined) {
                suspected[communityIP] = 0
            }
            suspected[communityIP] += 1
            if (suspected[communityIP] == MAX_HEALTH_CHECK_TRIES) {
                delete clusterToCommunity[services[communityIP].clusterIP]
                delete services[communityIP]
                delete suspected[communityIP]
            }
        }

    }
}, HEALTH_CHECK_INTERVAL)



app.listen(port, () => {
    console.log(`Discovery service listening on: ${port}`)
})