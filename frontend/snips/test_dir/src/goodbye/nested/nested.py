import random

def hello() -> str:
  # ? console log
  print('hello')
  # !!

  random.random()

  # HH-random 3
  if random.random() > 0.5:
    return 'yo'

  # HH-return 1 'hello'
  return 'hello'
