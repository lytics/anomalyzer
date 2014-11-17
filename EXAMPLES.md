
# Anomaly Detection Validation

As a sense check, we considered the results of our anomalyzer package for three different types of time series: cpu usage, membership, and seasonal data. First, we'll describe each algorithm in more detail and explain which scenarios it might be useful in. Then we'll detail what the algorithms return for these use cases.

## Algorithms In-depth

### CDF

Compares the differences between neighboring points in the active window to the cumulative distribution funtion of the differences between neighboring points in the reference window. This test is very sensitive to small fluctuations, and is perhaps best used on data with typically very little variance.

### Diff 

Considers the differences between neighboring points and ranks them, making this test a good measure of high volatility in the data. 

### High & Low Rank

Ranks the values themselves in the reference and active windows and computes how many times the sum over the permuted active window is less than (high rank) or greater than (low rank) the sum over the original active window. The former detects an increasing time series, and the latter detects a decreasing time series. For example, one might prefer to receive notifications about a decreasing audience size versus increasing audience size, making low rank preferred.

### Magnitude 

Considers the percentage difference between the averages of data in the active and reference windows. Since this probability is dynamically weighted, the result of the magnitude test is significantly down-weighted when it returns less than 80%, and is significantly up-weighted for results greater than 80%. Since this test considers the percentage change, it's a good measure of significant changes and is generally a good default.

Note that since it considers the averages over active and reference windows, keeping your active window smaller makes this test more sensitive. Additionally if the number of seasons specified is fairly long and your data has a history of prolonged increases or decreases, an increase in your active window might not be perceived as significant.

### Fence

Considers how close the average of the data in the active window is to an upper or lower bound. This test is also more sensitive for a smaller active window size. It is of course not applicable for data which does not lend itself to upper or lower bounds. For example, we'd like to know if CPU usage bottoms out at 0% or perhaps approaches 80%. But in terms of segment sizes, there is no natural upper or lower bound.

### Bootstrap KS

A hybrid of the CDF function and High Rank tests, this test compares the largest difference between the distributions of values in the active and reference windows to that difference after permuting the elements. If the permuted difference is smaller than the original difference, that signals that our initial active window distribution was anomalous. If the active window length is equal to a season, this test will detect seasonality in your time series.

## Use Case #1: CPU Usage

Detecting anomalies in CPU usage data could make debugging systems much easier, but since this data can be quite discontinuous, it's important to choose the right algorithms so that you aren't constantly being alerted. In particular, the behavior we'd like to detect as anomalies in CPU usage include: no longer receiving data (sharp falloff), maxxing out, and unusual spikes. We chose the active window length to be 2, which corresponded to a minute's worth of data. The number of seasons was 59, which meant my reference window encapsulated the past hour's worth of data. The upper bound was set to 80.0 and the lower bound was set to 0.0. Below we've shown the results of diff, cdf, magnitude, high rank, fence, and bootstrap ks tests on about a day's worth of CPU usage data. 

![cpu_usage_alltest2](https://cloud.githubusercontent.com/assets/6633242/4891879/6268d76c-63ac-11e4-97e0-cf0480630461.png)

The dark red areas correspond to regions where a test returned a probability greater than 90%. We decided not to show low rank here because we don't care that much about decreasing CPU usage. As well, sharp falloffs should be detected by magnitude and fence tests.

You can see that the CDF test was extremely sensitive to fluctuations, and makes it a bad test for this use case. Magnitude picked up the sharpest and largest peaks and drops in our data well. Our data never approached our 80% upper bound, but the fence test detected the dips to 0% that were made. We thought the ks and high rank tests did the best job of detecting unusual behavior, being a bit more informative than magnitude and fence. (Although, others may disagree and prefer the less detailed option.)

Below we've shown the weighted mean of these two tests. The regions in red correspond to an anomalous probability greater than 90% and 99%, on the first and second plot respectively.

![cpu_usage_weightedmean 90](https://cloud.githubusercontent.com/assets/6633242/4891900/944f2268-63ac-11e4-8b0c-ed80d8f5853a.png)

![cpu_usage_weightedmean 99](https://cloud.githubusercontent.com/assets/6633242/4891903/a09eadcc-63ac-11e4-9594-a82726d11b79.png)

You can see that by changing the threshold, we can get rid of some of the less important fluctuations.

## Use Case #2: Membership

We also considered an audience membership example because one might be interested in seeing if the number of users has significantly changed after a marketing campaign, for example. This data appears more continuous, very different from what CPU usage data looks like. In terms of audience membership, the tests we chose to run were diff, high rank, low rank, magnitude, cdf, and bootstrap ks. We again chose the active window length to be 2, which this time corresponded to four hours worth of data. And again the number of seasons was 59, which meant my reference window encapsulated the past 10 days worth of data.

![segments_alltest](https://cloud.githubusercontent.com/assets/6633242/4890831/88285d1a-63a2-11e4-8f77-f1ca8c74689d.png)

Again the bolded, red regions correspond to areas where a test returned higher than 90%. The diff test picked up some of the sharpest falloffs in the series like we'd expect. High rank detected the general peak in the data, and low rank detected the areas of decreasing membership, both as we'd expect. The magnitude test tells us that the increases/decreases in this timeseries are not particularly significant. The cdf test was extremely sensitive to fluctuations, again making it not ideal for this type of data. The ks test did a great job of picking out the peak in our data. 

On the one hand, one might be interested in seeing increases in membership. This would suggest considering the weighted mean of the ks and high rank tests. 

![segments_weightedmean peak](https://cloud.githubusercontent.com/assets/6633242/4892393/b7f6f808-63b1-11e4-9285-0b2dae7a1ba6.png)

Or someone might be more interested in the decreases in audience membership, suggesting the weighted mean of the low rank and diff tests.

![segments_weightedmean trough](https://cloud.githubusercontent.com/assets/6633242/4892397/c47b0218-63b1-11e4-82c9-4327d9370d84.png)

Again, the bold red in the above plots signal probabilities greater than 90%.

## Use Case #3: Seasonal Data

Lastly, we generated some seasonal data which could realistically stand in for CPU usage on a weekly basis, or gym membership over the course of a year, etc. In this case, it's important to recognize the changes that are significant with respect to prior seasons. The tests we chose to run were cdf, magnitude, highrank, lowrank, diff, and ks. We chose the active window to be equal to 10, the length of a season, and the number of seasons to be 2.

![seasonal_alltest1](https://cloud.githubusercontent.com/assets/6633242/4910185/1d136742-647c-11e4-9524-c36c7e3befba.png)

Again shown in red are probabilities greater than 90%. You can see above that the cdf, high rank, low rank, and diff tests are all a bit oversensitive to the fluctuations. None of them take into account the seasonal nature of this data. The ks test however does a good job of not over-reacting. 

![seasonal_alltest2](https://cloud.githubusercontent.com/assets/6633242/4910189/28aa1f6a-647c-11e4-9854-cf37f1cf5be3.png)

And for a sample with a bit more volatility, shown above, the ks test selects the atypical region out well.

