#! /usr/bin/Rscript

library(RJSONIO)
library(CausalImpact)

args <- commandArgs(trailingOnly = TRUE)
# the time series and time will be in string format

# unlist the time series and convert it to a number
y <- as.numeric(unlist(strsplit(args[1], ",")))

# convert time from a string to a number
time <- as.numeric(args[2])

pre.period <- c(1, (length(y)-time-1))
post.period <- c(length(y)-time, length(y))

impact <- CausalImpact(y, pre.period, post.period)

loweravg <- impact$summary$RelEffect.lower[1]
upperavg <- impact$summary$RelEffect.upper[1]
tailareaprob <- impact$summary$p[1]

results <- list(lower = loweravg, upper = upperavg, p = tailareaprob)
cat(toJSON(results, collapse = ""))