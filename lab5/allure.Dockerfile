FROM node:22-alpine AS cli
RUN npm install --prefix /opt/allure allure-commandline@2.34.1
FROM eclipse-temurin:17-jre
COPY --from=cli /opt/allure/node_modules/allure-commandline/dist /opt/allure
COPY allure-overview.html /opt/allure-overview.html
ENTRYPOINT ["sh", "-ec", "/opt/allure/bin/allure generate /results/allure-results --single-file -o /results/allure-report \"$@\" && sed -i '/^<head>/r /opt/allure-overview.html' /results/allure-report/index.html", "report"]
