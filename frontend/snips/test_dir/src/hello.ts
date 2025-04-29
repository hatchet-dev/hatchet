// import * from "@hatchet"

function hello() {
  // > console error
  console.error('hello');
  // !!

  // > console log
  console.log('hello');
  // !!

  // > multiple lines
  console.log('hello');
  console.log('world');
  // !!

  // HH-random 3
  if (Math.random() > 0.5) {
    return 'yo';
  }

  // HH-return 1 'hello'
  return 'hello';
}

export default hello;
