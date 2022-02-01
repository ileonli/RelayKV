FROM ubuntu

COPY bin/* /local/bin/

ENV LEADER 0
ENV CLUSTER 9870

CMD /local/bin/server $LEADER $CLUSTER