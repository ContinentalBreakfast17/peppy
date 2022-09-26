# this can be used for testing lambda triggers
import json

def handler(event, context): 
    print(json.dumps(event))