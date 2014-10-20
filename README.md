
# Anomalyzer

This repository contains two different projects for determining anomalous behavior in a time series.

## Contents

The first project, **Anomalyzer**, implements five different statistical tests each yielding probabilities of anomalous behavior. A weighted mean of the probabilities from each chosen statistical test is returned. A more detailed description with information regarding configuration of this project can be found [here] (https://github.com/lytics/anomalyzer/tree/master/anomalyzer). 

Anomalyzer can be used to consider any type of time series data (i.e. group membership, number of url visits, etc). We considered the CPU usage of one of our servers in **Influx Client**. This information was exported to an InfluxDB. We queried this database and applied Anomalyzer to the obtained data. Beyond our configuration choices for our Anomalyzer, the code can be used for any time series data from InfluxDB. More information about that application can be found [here] (https://github.com/lytics/anomalyzer/tree/master/anomalyzer/influxclient).

The second project, **Causal Impact**, accesses [Google's recently developed R library] (https://github.com/google/CausalImpact) for determining causal inference. This project uses Go's "os/exec" package to run an RScript with this library. A more detailed description and information regarding set up can be found [here] (https://github.com/lytics/anomalyzer/tree/master/causalimpact).
