#! /usr/bin/Rscript

library(RJSONIO)
library(CausalImpact)

args <- commandArgs(trailingOnly = TRUE)
# y and t will be in string format

y <- as.numeric(unlist(strsplit(args[1], ",")))

time <- as.numeric(args[2])

pre.period <- c(1, (time-1))
post.period <- c(time, length(y))

impact <- CausalImpact(y, pre.period, post.period)

loweravg <- impact$summary$RelEffect.lower[1]
upperavg <- impact$summary$RelEffect.upper[1]
tailareaprob <- impact$summary$p[1]

results <- list(lower = loweravg, upper = upperavg, p = tailareaprob)
cat(toJSON(results, collapse = ""))