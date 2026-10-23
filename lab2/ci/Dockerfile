FROM docker:29-cli AS cli
FROM jenkins/jenkins:lts-jdk21
USER root
RUN apt-get update && apt-get install -y --no-install-recommends python3 make && rm -rf /var/lib/apt/lists/*
COPY --from=cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=cli /usr/local/libexec/docker/cli-plugins/docker-compose /usr/local/lib/docker/cli-plugins/docker-compose
RUN jenkins-plugin-cli --plugins workflow-aggregator junit pipeline-stage-view timestamper
COPY init.groovy /usr/share/jenkins/ref/init.groovy.d/lab2.groovy
USER jenkins
