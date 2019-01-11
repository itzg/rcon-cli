# Creating a release

Tag the source repository and push the tag. The CircleCI configuration includes a `release`
workflow that will take care of invoking goreleaser. 