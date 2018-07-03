#!/bin/bash
 ps -a | grep ont | awk '{print $1}' | xargs kill
