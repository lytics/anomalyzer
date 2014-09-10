
# Anomalizer

Probability-based anomaly detection in Go.

![anomaly2](https://cloud.githubusercontent.com/assets/6633242/4197767/d564fd66-37ee-11e4-9093-695227ebe217.png)
![anomaly4](https://cloud.githubusercontent.com/assets/6633242/4197802/5674a906-37ef-11e4-94b1-c1dd808363d3.png)

This code returns the probability that a given time series contains anomalous behavior using four different statistical tests. At the beginning of the code, the length of an "active" window (the window over which we want to investigate anomalous behavior) is specified. The data that comes before the active window is considered the "reference" for our test

The tests are:

	1) "Prob" -> Calculates the percent difference between the averages of the reference and active data sets.

	2) "Rank" -> Implements a boostrap permutation test.

	3) "DiffCDF" -> Compares the cumulative distribution functions of the active and reference windows.

	4) "Bounds" -> Flags data that is moving close to an upper or lower bound (which can be specified at the top of the code).

Additionally, a weighted sum is implemented. It is suggested that the weights be changed to the user's liking and/or depending on the lengths of the reference window considered.

After considering reference windows of different lengths, it appears that the Rank test is slightly more sensitive over a longer reference window, Prob over a shorter reference window, and DiffCDF is generally applicable when both shorter and larger reference windows are considered.

(Kilmogorov-Smirnov is also implemented in the code, but its result is not shown or considered part of the weighted sum.)
