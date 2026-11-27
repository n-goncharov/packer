# Packer

## Getting Started

### Prerequisites

- `Go 1.27.4+`
### Running

```sh
go run cmd/main/main.go
```


## Project Plan
- [x] Refactor the current project architecture and code (specifically, address current TODOs).
  - [x] Layer interfaces and folder/package structure and naming, including provisions for metaheuristics.
  - [x] Implementations.
- [ ] Save result data (strips, algo used, FitVal, PackedRects, execution time) of each run.
- [ ] Add the generation of an HTML file with a packing visualization next to result data.
- [ ] Add automatic browser window opening to display that HTML.
- [ ] Add random input data generation.
- [ ] Read input (rects and strips parameters from file from json-file) and configs (algo and its params from yaml-file).
- [ ] Add another queue-based heuristics: NFDH, FFDH, BFDH, BL.
- [ ] Add CLI-flags to run different configs.
- [ ] Add an automatic generation of rectangle sets.
- [ ] Add a genetic algorithm.
- [ ] Add a plot showing the objective function's progression and other useful metaheuristic result data — specifically tailored to the algorithm (e.g., a population diversity plot for the GA).

### Extra:
- [ ] Add a tree-based heuristic (oriented trees).
- [ ] Add guillotine/non-guillotine cutting methods to generate rectangles (the optimum will be known).
- [ ] Add another metaheuristic (e.g., simulated annealing, ant colony).
