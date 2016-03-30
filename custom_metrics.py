import random

value = random.randint(101, 200)
metric = "requests.per_second:{0}|gauge".format(value)

print metric
