# Use Debian or Ubuntu as the base image
FROM debian:latest

# Set environment variables to avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Update system and install necessary tools
RUN apt update && apt upgrade -y && \
    apt install -y sudo python3 python3-pip vim curl git unzip && \
    apt clean

# Define a username as an argument (can be overridden at build time)
ARG USERNAME=newuser
ENV USERNAME=${USERNAME}

# Create a new user and grant sudo privileges
RUN useradd -m -s /bin/bash "$USERNAME" && \
    echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set environment variable for user home directory
ENV LOCATION=/home/$USERNAME

# Copy task and break lists into the container
COPY todo.csv ${LOCATION}/todo.csv
COPY break.csv ${LOCATION}/break.csv

# Copy the setup script and make it executable
COPY setup.sh ${LOCATION}/setup.sh
RUN chmod +x ${LOCATION}/setup.sh

# Set user to "$USERNAME" for when the container is run
USER $USERNAME

# Run the setup script when the container starts
CMD ["bash", "-c", "$LOCATION/setup.sh $LOCATION"]

