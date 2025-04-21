// first we need to create a stage
var stage = new Konva.Stage({
	container: 'container', // id of container <div>
	width: 1000,
	height: 1000,
});

// then create layer
var layer = new Konva.Layer();

const dim = 40
const inc = 1000 / dim

for (var i = 0; i <= dim; i++) {

	const colLine = new Konva.Line({
		points: [i * inc, 0, i * inc, 1000],
		stroke: 'red',
		strokeWidth: 2,
	});
	layer.add(colLine)
	const rowLine = new Konva.Line({
		points: [0, i * inc, 1000, i * inc],
		stroke: 'red',
		strokeWidth: 2,
	});
	layer.add(rowLine)

}


// add the layer to the stage
stage.add(layer);
